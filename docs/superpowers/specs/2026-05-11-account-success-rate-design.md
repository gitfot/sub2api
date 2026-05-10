# 账号请求成功率统计设计

## 背景

当前项目已经有两类原始日志：

- `usage_logs`：成功请求
- `ops_error_logs`：失败请求

用户希望补齐类似 CLIProxyAPI `/v0/management/api-key-usage` 的账号级成功/失败统计，但统计口径要保持“原始成功/失败”，不做 SLA 过滤。

本设计只使用当前保留窗口内的数据，避免新增长期历史存储。

## 目标

- 仪表盘新增“请求成功率”卡片，展示今日成功率/历史成功率。
- 仪表盘新增账号请求成功率趋势图，默认只看当前正常账号。
- 账号管理列表新增“成功率”列，展示每个账号的历史总请求成功率。
- 全部统计基于原始成功/失败，不改变现有请求处理逻辑。
- 在线查询不扫原始日志，保持性能稳定。

## 非目标

- 不新增“全生命周期永久历史”。
- 不做 SLA 成功率口径。
- 不恢复历史账号状态快照。
- 不改请求写入链路，只改统计与展示链路。

## 现有约束

- `usage_logs` 默认保留 90 天。
- 当前仪表盘统计使用 `snapshot-v2`，不是旧的 `dashboard/stats` 主路径。
- 当前“正常账号”语义已存在，等于 `status=active && schedulable=true`。
- `ops_error_logs.status_code >= 400` 视为失败。
- `is_count_tokens = true` 的探测请求不计入成功率统计。

## 方案

采用方案 C：新增小时级以上的账号请求预聚合表，但底层粒度固定为 10 分钟。

原因：

- 能对齐 CLIProxyAPI 近似的窗口感。
- 读取时不需要扫原始日志。
- 10 分钟粒度足够支撑趋势图和列表汇总。

## 数据模型

新增聚合表 `account_request_stats_10m`。

建议字段：

- `bucket_start`
- `account_id`
- `success_count`
- `failed_count`
- `request_count`
- `computed_at`

其中：

- `request_count = success_count + failed_count`
- `success_rate` 不落库，查询时计算

建议索引：

- `(bucket_start, account_id)` 唯一索引
- `(account_id, bucket_start desc)` 普通索引
- `(bucket_start desc)` 普通索引

## 聚合规则

### 成功

来自 `usage_logs`，按 `account_id` 与 10 分钟桶聚合。

### 失败

来自 `ops_error_logs`，只统计 `status_code >= 400` 的记录。

### 过滤

- 排除 `is_count_tokens = true`
- `account_id IS NULL` 的失败日志不进入账号级统计

### 时间边界

- 使用 UTC 统一聚合桶
- 读取端按现有仪表盘的时间边界逻辑展示

## 后台作业

复用现有 `dashboard_aggregation_service` 的定时作业，不另起一套调度器。

原因：

- 已有 watermark
- 已有回填
- 已有保留清理
- 已有重算与容错框架

建议行为：

- 每次运行重算最近一段重叠窗口，吸收晚到日志
- 继续使用 watermark 推进增量聚合
- 新表保留周期跟 `usage_logs_days=90` 对齐
- 清理时同步删除新聚合表中过期桶

## API 设计

### 1. 仪表盘快照

扩展 `GET /api/v1/admin/dashboard/snapshot-v2`

在 `stats` 中新增：

- `today_success_count`
- `today_failed_count`
- `today_success_rate`
- `history_success_rate`

口径：

- `today_success_rate = 今日成功数 / (今日成功数 + 今日失败数)`
- `history_success_rate = 保留窗口内成功数 / (成功数 + 失败数)`

这里按所有账号请求统计，不过滤 normal 账号。

### 2. 趋势图

新增 `GET /api/v1/admin/dashboard/account-success-rate-trend`

查询参数：

- `start_date` / `end_date`
- `granularity=10m|1h|1d`

响应建议：

```json
{
  "bucket": "10m",
  "computed_at": "2026-05-11T10:00:00Z",
  "stale": false,
  "partial": false,
  "points": [
    {
      "bucket_start": "2026-05-11T09:00:00Z",
      "success_count": 120,
      "failed_count": 8,
      "request_count": 128,
      "success_rate": 93.75,
      "accounts": [
        {
          "account_id": 101,
          "account_name": "claude-01",
          "success_count": 40,
          "failed_count": 2,
          "request_count": 42,
          "success_rate": 95.24
        }
      ]
    }
  ]
}
```

趋势图只统计当前正常账号，且正常账号按“查询时当前状态”判断，不回溯历史状态。

### 3. 账号批量成功率

新增 `POST /api/v1/admin/accounts/success-rate/batch`

请求体：

```json
{
  "account_ids": [101, 102, 103]
}
```

响应建议：

```json
{
  "stats": {
    "101": {
      "success_count": 12340,
      "failed_count": 120,
      "request_count": 12460,
      "success_rate": 99.04
    }
  }
}
```

`request_count = 0` 时返回 `success_rate = null`，前端显示 `--`。

## 前端设计

### 仪表盘

- 替换第一张 API Key 卡片为“请求成功率”
- 卡片展示：
  - 今日成功率
  - 历史成功率
- 新增一个柱状图卡片：
  - x 轴时间
  - y 轴成功率
  - 只展示正常账号
  - tooltip 展示账号明细
- 继续复用现有日期范围控件
- 成功率图可单独支持 `10m / 1h / 1d`

### 账号列表

- 新增一列“成功率”
- 推荐位置：放在今日统计之后、其他运维列之前
- 单元格展示：
  - 主值：成功率
  - 次值：成功 / 失败
  - 无请求时显示 `--`

### 页面行为

- 列表当前页加载完成后，批量拉取当前页账号成功率
- 翻页、刷新、筛选时重新拉取
- 接口失败时不阻塞表格，只显示降级态

## 异常处理

- 如果聚合表暂无数据：
  - 卡片显示 `--`
  - 图表空态
  - 列表列显示 `--`
- 不在在线查询里回扫原始日志兜底
- 允许短暂数据滞后，并通过 `computed_at / stale / partial` 对外提示

## 性能边界

- 仪表盘与列表不得在正常加载路径上扫描 `usage_logs` 或 `ops_error_logs`
- 趋势图与列表只读聚合表
- 10 分钟桶是基础粒度，1 小时/1 天通过聚合桶再汇总

## 测试

### 后端

- 聚合表建表与索引测试
- 10 分钟 upsert 测试
- 成功/失败口径测试
- dashboard snapshot 新字段测试
- 趋势接口测试
- batch 接口测试
- 保留清理测试

### 前端

- Dashboard 成功率卡片渲染测试
- Dashboard 趋势图加载与空态测试
- Accounts 列成功率列渲染测试
- batch 接口失败降级测试

## 验收标准

- 仪表盘能看到请求成功率卡片
- 仪表盘能看到 normal 账号成功率趋势图
- 账号列表能看到每个账号的历史成功率
- 所有成功率都按原始成功/失败计算
- 在线页面加载不依赖原始日志全表扫描

