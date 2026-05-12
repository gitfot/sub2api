# 上游错误归因与详情展示优化设计

## 背景

当前 `POST /v1/chat/completions` 命中上游后，如果上游返回类似下面的错误：

```json
{
  "error": {
    "message": "openai_error",
    "type": "invalid_request_error"
  }
}
```

系统会出现这些现象：

- 仪表盘“请求错误”计数增加。
- 账号成功率失败数增加。
- 运维监控错误列表出现一条 400 请求错误。
- “上游错误”卡片不增加。
- 请求错误详情弹窗不能直观看出具体是哪个上游账号产生的错误。

根因是当前错误分类把 `invalid_request_error` 固定归为 `request/client`，而“上游错误”统计只看 `error_owner = 'provider'`。因此，即使后端已经捕获了 `upstream_status_code`、`upstream_error_message`、`upstream_errors` 和 `account_id`，统计与前端展示仍会把这类错误当成客户端请求错误。

## 目标

- 上游实际返回 4xx/5xx 时，“上游错误”卡片必须计数。
- 客户端最终收到 4xx/5xx 时，“请求错误”卡片仍然计数。
- 一条错误记录可以同时表达“客户端请求失败”和“上游账号返回错误”两个事实。
- 请求错误详情弹窗必须能直接看到产生错误的上游账号、上游状态码、上游 request id 和上游错误摘要。
- 账号成功率失败数继续按客户端请求失败统计，不改变当前失败口径。

## 非目标

- 不改变对外 API 错误响应格式。
- 不把所有 `invalid_request_error` 都当成上游错误。
- 不重做 Ops 错误表结构的大迁移。
- 不改变 `count_tokens` 等现有过滤规则。
- 不改变账号调度、failover、限流处理策略。

## 口径定义

采用双口径：

### 请求错误口径

客户端最终响应 `status_code >= 400` 即为请求错误。

这类错误继续进入：

- 请求错误卡片
- 请求错误列表
- 账号成功率失败数

### 上游错误口径

只要请求实际打到上游，且上游返回 `upstream_status_code >= 400`，就计为上游错误。

这包括：

- 400
- 401
- 403
- 404
- 422
- 429
- 500+
- 529

不再因为响应体的 `error.type` 是 `invalid_request_error` 就把归因固定为客户端。

## 后端归因规则

`OpsErrorLoggerMiddleware` 创建错误记录时，应先判断是否存在有效上游错误上下文。

有效上游错误上下文定义为：

- `OpsUpstreamStatusCodeKey >= 400`，或
- `OpsUpstreamErrorsKey` 中存在 `upstream_status_code >= 400` 的事件。

当存在有效上游错误上下文时：

- `error_owner = provider`
- `error_source = upstream_http`
- `error_phase = upstream`

同时保留客户端最终响应字段：

- `status_code`：客户端最终 HTTP 状态码，例如 400。
- `error_type`：客户端最终错误类型，例如 `invalid_request_error`。
- `error_message`：客户端最终错误消息或透传后的消息。
- `error_body`：客户端最终响应体。

当不存在有效上游错误上下文时，继续使用现有分类：

- 本地参数错误：`request/client`
- API Key 错误：`auth/client`
- 路由失败：`routing/platform`
- 内部异常：`internal/platform`

## 统计影响

### 请求错误

请求错误统计继续基于：

- `status_code >= 400`

因此上游 400 仍会让请求错误 +1。

### 上游错误

上游错误统计继续基于现有 provider 口径：

- `error_owner = 'provider'`
- `NOT is_business_limited`
- `COALESCE(upstream_status_code, status_code, 0)` 按当前 429/529 拆分规则统计。

因为上游 4xx/5xx 会被归因为 `provider`，所以它会进入上游错误统计。

### 账号成功率

账号成功率失败数继续基于：

- `ops_error_logs.status_code >= 400`
- `account_id IS NOT NULL`
- `is_count_tokens = FALSE`

不按 `error_owner` 过滤，避免改变当前成功率口径。

## 详情弹窗设计

请求错误详情页必须同时展示客户端结果与上游归因。

### 请求结果区域

展示：

- 客户端状态码：`status_code`
- 客户端错误类型：`error_type`
- 客户端错误消息：`error_message`
- 入站 endpoint：`inbound_endpoint`
- 请求模型：`requested_model`
- 上游模型：`upstream_model`

### 上游归因区域

只要存在 `account_id`、`upstream_status_code`、`upstream_error_message` 或 `upstream_errors`，就展示上游归因区域。

展示：

- 上游账号名与账号 ID
- 上游平台
- 上游 endpoint
- 上游状态码
- 上游 request id
- 上游错误摘要

用户信息不应被上游账号展示替代。详情页应同时保留用户信息和上游账号信息。

### 上游尝试列表

详情页应优先解析当前记录的 `upstream_errors` JSON，而不是只依赖关联查询接口。

每个事件展示：

- 账号名与账号 ID
- 上游状态码
- 上游 request id
- `kind`
- `message`
- `detail`
- 必要时展示脱敏后的上游响应摘要。

如果 `upstream_errors` 为空，但单字段 `upstream_status_code`、`upstream_error_message`、`account_id` 存在，则前端生成一条 fallback 摘要，避免显示“无数据”。

现有的 `GET /admin/ops/request-errors/:id/upstream-errors` 可作为补充来源，但不应是详情页唯一的上游信息来源。

## 数据流

1. Handler 选中账号后写入 `account_id` 上下文。
2. Gateway service 调用上游。
3. 上游返回 4xx/5xx。
4. Gateway service 设置：
   - `OpsUpstreamStatusCodeKey`
   - `OpsUpstreamErrorMessageKey`
   - `OpsUpstreamErrorDetailKey`
   - `OpsUpstreamErrorsKey`
5. Gateway service 按兼容协议向客户端写出最终错误响应。
6. Ops error logger 捕获客户端错误响应。
7. Ops error logger 根据上游上下文把归因字段设为 provider/upstream。
8. 运维面板同时在请求错误与上游错误口径中体现这条记录。

## 测试计划

### 后端

新增或调整 Ops error logger 测试：

- 上游返回 400 `invalid_request_error`：
  - `status_code = 400`
  - `error_type = invalid_request_error`
  - `upstream_status_code = 400`
  - `error_owner = provider`
  - `error_source = upstream_http`
  - `error_phase = upstream`
  - `account_id` 被记录

- 本地请求校验失败，例如缺少 `model`：
  - `status_code = 400`
  - `error_type = invalid_request_error`
  - 无 `upstream_status_code`
  - `error_owner = client`
  - 不计入上游错误。

- Dashboard 上游错误统计：
  - provider 400 进入上游错误总数。
  - client 400 不进入上游错误总数。

- 账号成功率聚合：
  - provider 400 仍进入账号失败数。

### 前端

新增或调整详情弹窗测试：

- 请求错误详情中存在 `account_id` 与上游上下文时，显示上游账号。
- 请求错误详情同时显示用户信息和上游账号信息。
- `upstream_errors` 有事件时，展示上游尝试列表。
- `upstream_errors` 为空但单字段上游信息存在时，展示 fallback 摘要。

## 验收标准

- 上游返回 400 `invalid_request_error` 时：
  - 请求错误卡片 +1。
  - 上游错误卡片 +1。
  - 对应账号成功率失败数 +1。
  - 请求错误详情能看到上游账号、上游状态码、上游 request id 和上游错误摘要。
- 本地参数校验失败时：
  - 请求错误卡片 +1。
  - 上游错误卡片不增加。
  - 详情页不误展示上游账号。
- 现有 API 错误响应格式保持兼容。

