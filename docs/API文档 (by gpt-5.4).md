# API KEY 对外接口文档

本文档整理了当前项目中所有基于 API KEY 对外开放的网关接口，适合作为调用方接入参考。

说明：

- 本文档只覆盖 **API KEY 鉴权** 的外部接口，不包含后台管理或前端登录后使用的 JWT 接口。
- 项目本质上是多协议网关，部分接口会根据 API Key 所属分组的 `platform` 自动路由到 OpenAI、Anthropic、Gemini 或 Antigravity。
- 以下字段说明以项目当前显式校验和常用协议字段为主；对于兼容代理接口，未列出的透传字段通常仍会按上游协议转发。

---

## 目录

- [认证方式](#认证方式)
- [接口总览](#接口总览)
- [别名路径](#别名路径)
- [OpenAI / Anthropic 兼容接口](#openai--anthropic-兼容接口)
- [Gemini 原生兼容接口](#gemini-原生兼容接口)
- [Antigravity 专用接口](#antigravity-专用接口)

---

## 认证方式

### 通用网关接口认证

适用于：

- `/v1/*`
- `/responses`
- `/chat/completions`
- `/images/*`
- `/backend-api/codex/*`
- `/antigravity/*`

推荐请求头：

```http
Authorization: Bearer sk-xxxx
Content-Type: application/json
```

兼容请求头：

- `x-api-key: sk-xxxx`
- `x-goog-api-key: sk-xxxx`

注意：

- 普通网关接口 **不再支持** `?key=` 或 `?api_key=` 传参。
- API Key 还会经过用户状态、IP 白名单/黑名单、余额/订阅/额度检查。

### Gemini 原生接口认证

适用于：

- `/v1beta/*`
- `/antigravity/v1beta/*`

推荐请求头：

```http
x-goog-api-key: sk-xxxx
Content-Type: application/json
```

也兼容：

- `Authorization: Bearer sk-xxxx`
- `x-api-key: sk-xxxx`

仅 Gemini 风格接口兼容：

- `?key=sk-xxxx`

---

## 接口总览

| 协议分组 | 方法 | 路径 | 说明 |
|----------|------|------|------|
| Anthropic/OpenAI 兼容 | `POST` | `/v1/messages` | Anthropic Messages 接口 |
| Anthropic/OpenAI 兼容 | `POST` | `/v1/messages/count_tokens` | 估算 Messages 请求输入 token |
| OpenAI/Anthropic 兼容 | `GET` | `/v1/models` | 获取当前分组可用模型列表 |
| OpenAI/Anthropic 兼容 | `GET` | `/v1/usage` | 获取当前 API Key 的额度/余额/统计信息 |
| OpenAI 兼容 | `POST` | `/v1/responses` | OpenAI Responses 接口 |
| OpenAI 兼容 | `POST` | `/v1/responses/*subpath` | Responses 子路径，如 `/v1/responses/compact` |
| OpenAI 兼容 | `GET` | `/v1/responses` | Responses WebSocket 入口 |
| OpenAI 兼容 | `POST` | `/v1/chat/completions` | OpenAI Chat Completions 接口 |
| OpenAI 兼容 | `POST` | `/v1/images/generations` | OpenAI 图片生成接口 |
| OpenAI 兼容 | `POST` | `/v1/images/edits` | OpenAI 图片编辑接口 |
| Gemini 原生 | `GET` | `/v1beta/models` | Gemini 模型列表 |
| Gemini 原生 | `GET` | `/v1beta/models/:model` | Gemini 单模型详情 |
| Gemini 原生 | `POST` | `/v1beta/models/*modelAction` | Gemini `generateContent`/`streamGenerateContent` |
| Antigravity | `GET` | `/antigravity/models` | Antigravity 模型列表 |
| Antigravity | `POST` | `/antigravity/v1/messages` | 强制 Antigravity 的 Messages 接口 |
| Antigravity | `POST` | `/antigravity/v1/messages/count_tokens` | 强制 Antigravity 的 token 估算接口 |
| Antigravity | `GET` | `/antigravity/v1/models` | 强制 Antigravity 的模型列表 |
| Antigravity | `GET` | `/antigravity/v1/usage` | 强制 Antigravity 的用量/额度查询 |
| Antigravity | `GET` | `/antigravity/v1beta/models` | Antigravity 的 Gemini 风格模型列表 |
| Antigravity | `GET` | `/antigravity/v1beta/models/:model` | Antigravity 的 Gemini 风格单模型详情 |
| Antigravity | `POST` | `/antigravity/v1beta/models/*modelAction` | Antigravity 的 Gemini 风格内容生成接口 |

---

## 别名路径

以下路径与规范路径功能等价，通常用于兼容不同 SDK 或客户端：

| 别名路径 | 对应规范路径 | 说明 |
|----------|--------------|------|
| `POST /responses` | `POST /v1/responses` | 无 `/v1` 前缀别名 |
| `POST /responses/*subpath` | `POST /v1/responses/*subpath` | Responses 子路径别名 |
| `GET /responses` | `GET /v1/responses` | Responses WebSocket 别名 |
| `POST /chat/completions` | `POST /v1/chat/completions` | 无 `/v1` 前缀别名 |
| `POST /images/generations` | `POST /v1/images/generations` | 无 `/v1` 前缀别名 |
| `POST /images/edits` | `POST /v1/images/edits` | 无 `/v1` 前缀别名 |
| `POST /backend-api/codex/responses` | `POST /v1/responses` | Codex 兼容入口 |
| `POST /backend-api/codex/responses/*subpath` | `POST /v1/responses/*subpath` | Codex 兼容子路径 |
| `GET /backend-api/codex/responses` | `GET /v1/responses` | Codex WebSocket 入口 |

---

## OpenAI / Anthropic 兼容接口

### `POST /v1/messages`

接口名：Anthropic Messages

用途：

- 使用 Anthropic Messages 协议发起对话。
- 当 API Key 所属分组平台为 OpenAI 时，项目会自动进行协议转换和转发。

#### 入参说明

请求头：

```http
Authorization: Bearer sk-xxxx
Content-Type: application/json
```

请求体核心字段：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `model` | `string` | 是 | 模型名 |
| `max_tokens` | `number` | 是 | 最大输出 token 数 |
| `messages` | `array` | 是 | 对话消息数组 |
| `system` | `string / array` | 否 | 系统提示词 |
| `tools` | `array` | 否 | 工具定义 |
| `stream` | `boolean` | 否 | 是否流式返回 |
| `temperature` | `number` | 否 | 温度参数 |
| `top_p` | `number` | 否 | top-p 采样 |
| `stop_sequences` | `array` | 否 | 停止序列 |
| `metadata` | `object` | 否 | 透传元数据 |

`messages[].role` 常见值：

- `user`
- `assistant`

`messages[].content` 可为：

- 纯文本字符串
- 内容块数组，例如 `text`、`image`、`tool_result`

#### 入参示例

```json
{
  "model": "claude-sonnet-4-20250514",
  "max_tokens": 1024,
  "system": "你是一个简洁的中文助手。",
  "messages": [
    {
      "role": "user",
      "content": "请用三句话介绍 Sub2API。"
    }
  ],
  "stream": false
}
```

#### 出参说明

非流式响应核心字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | `string` | 消息 ID |
| `type` | `string` | 固定为 `message` |
| `role` | `string` | 固定为 `assistant` |
| `content` | `array` | 返回内容块 |
| `model` | `string` | 实际响应模型 |
| `stop_reason` | `string` | 停止原因 |
| `usage.input_tokens` | `number` | 输入 token |
| `usage.output_tokens` | `number` | 输出 token |

#### 出参示例

```json
{
  "id": "msg_01abcxyz",
  "type": "message",
  "role": "assistant",
  "content": [
    {
      "type": "text",
      "text": "Sub2API 是一个多协议 AI 网关。它可以用统一的 API KEY 对接多种上游账号和模型。它还支持 OpenAI、Anthropic、Gemini 等协议兼容。"
    }
  ],
  "model": "claude-sonnet-4-20250514",
  "stop_reason": "end_turn",
  "usage": {
    "input_tokens": 28,
    "output_tokens": 49,
    "cache_creation_input_tokens": 0,
    "cache_read_input_tokens": 0
  }
}
```

---

### `POST /v1/messages/count_tokens`

接口名：Anthropic Count Tokens

用途：

- 估算 `/v1/messages` 请求输入 token。
- 不记录实际使用量。

#### 入参说明

请求体与 `/v1/messages` 基本一致，常用字段如下：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `model` | `string` | 是 | 模型名 |
| `messages` | `array` | 是 | 对话消息数组 |
| `system` | `string / array` | 否 | 系统提示词 |
| `tools` | `array` | 否 | 工具定义 |

#### 入参示例

```json
{
  "model": "claude-sonnet-4-20250514",
  "system": "你是一个代码助手。",
  "messages": [
    {
      "role": "user",
      "content": "帮我生成一个 Go HTTP 服务样例。"
    }
  ]
}
```

#### 出参说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `input_tokens` | `number` | 估算得到的输入 token 数 |

#### 出参示例

```json
{
  "input_tokens": 42
}
```

---

### `GET /v1/models`

接口名：模型列表

用途：

- 返回当前 API Key 分组下可用的模型列表。
- 如分组配置了模型白名单，则优先返回白名单。

#### 入参说明

无请求体。

| 参数 | 位置 | 必填 | 说明 |
|------|------|------|------|
| 无 | - | - | 仅需 API KEY 请求头 |

#### 入参示例

```bash
curl -X GET "http://localhost:8080/v1/models" \
  -H "Authorization: Bearer sk-xxxx"
```

#### 出参说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `object` | `string` | 固定为 `list` |
| `data` | `array` | 模型列表 |
| `data[].id` | `string` | 模型 ID |
| `data[].type` | `string` | 一般为 `model` |
| `data[].display_name` | `string` | 展示名称 |
| `data[].created_at` | `string` | 创建时间占位值 |

#### 出参示例

```json
{
  "object": "list",
  "data": [
    {
      "id": "gpt-5.4-mini",
      "type": "model",
      "display_name": "gpt-5.4-mini",
      "created_at": "2024-01-01T00:00:00Z"
    },
    {
      "id": "gpt-5.4",
      "type": "model",
      "display_name": "gpt-5.4",
      "created_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

---

### `GET /v1/usage`

接口名：API Key 用量与额度查询

用途：

- 查询当前 API Key 的余额、配额、速率限制和统计数据。
- 即使 Key 已过期或额度耗尽，该接口仍允许查询自身状态。

#### 入参说明

查询参数：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `start_date` | `string` | 否 | 开始日期，格式 `YYYY-MM-DD` |
| `end_date` | `string` | 否 | 结束日期，格式 `YYYY-MM-DD` |

#### 入参示例

```bash
curl -X GET "http://localhost:8080/v1/usage?start_date=2026-05-01&end_date=2026-05-12" \
  -H "Authorization: Bearer sk-xxxx"
```

#### 出参说明

该接口分两类响应：

- `quota_limited`：当前 Key 配置了总额度或速率限制
- `unrestricted`：按订阅额度或钱包余额返回

核心字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| `mode` | `string` | `quota_limited` 或 `unrestricted` |
| `isValid` | `boolean` | 当前 Key 是否可视为有效 |
| `status` | `string` | Key 状态，配额模式下常见 |
| `remaining` | `number` | 剩余额度或余额 |
| `unit` | `string` | 当前为 `USD` |
| `quota` | `object` | Key 总额度信息 |
| `rate_limits` | `array` | 速率限制窗口信息 |
| `subscription` | `object` | 订阅模式下的限额信息 |
| `balance` | `number` | 钱包模式下的余额 |
| `usage` | `object` | today/total 聚合统计 |
| `model_stats` | `array` | 按模型聚合统计 |

#### 出参示例

```json
{
  "mode": "quota_limited",
  "isValid": true,
  "status": "active",
  "remaining": 18.4,
  "unit": "USD",
  "quota": {
    "limit": 20,
    "used": 1.6,
    "remaining": 18.4,
    "unit": "USD"
  },
  "rate_limits": [
    {
      "window": "1d",
      "limit": 100,
      "used": 12,
      "remaining": 88,
      "window_start": "2026-05-12T00:00:00+08:00",
      "reset_at": "2026-05-13T00:00:00+08:00"
    }
  ],
  "usage": {
    "today": {
      "requests": 12,
      "input_tokens": 4321,
      "output_tokens": 2876,
      "cache_creation_tokens": 0,
      "cache_read_tokens": 0,
      "total_tokens": 7197,
      "cost": 0.82,
      "actual_cost": 0.79
    },
    "total": {
      "requests": 58,
      "input_tokens": 22310,
      "output_tokens": 16442,
      "cache_creation_tokens": 0,
      "cache_read_tokens": 0,
      "total_tokens": 38752,
      "cost": 4.46,
      "actual_cost": 4.32
    },
    "average_duration_ms": 1240,
    "rpm": 2,
    "tpm": 950
  }
}
```

---

### `POST /v1/responses`

接口名：OpenAI Responses

用途：

- 使用 OpenAI Responses 协议进行对话、工具调用和多模态输入。
- 是项目 OpenAI 兼容能力的主入口之一。

#### 入参说明

请求体核心字段：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `model` | `string` | 是 | 模型名 |
| `input` | `string / array` | 是 | 输入内容 |
| `instructions` | `string` | 否 | 系统指令 |
| `stream` | `boolean` | 否 | 是否流式 |
| `tools` | `array` | 否 | 工具定义 |
| `max_output_tokens` | `number` | 否 | 最大输出 token |
| `temperature` | `number` | 否 | 温度参数 |
| `top_p` | `number` | 否 | top-p 采样 |
| `reasoning` | `object` | 否 | 推理强度设置 |
| `text` | `object` | 否 | 文本详细度设置 |
| `tool_choice` | `string / object` | 否 | 工具选择策略 |
| `service_tier` | `string` | 否 | 服务层级 |
| `prompt_cache_key` | `string` | 否 | 会话粘性缓存键 |
| `previous_response_id` | `string` | 否 | 上一轮响应 ID，仅部分模式支持 |

`input` 常见写法：

- 字符串
- Role message 数组
- `function_call_output` 数组

#### 入参示例

```json
{
  "model": "gpt-5.4-mini",
  "instructions": "你是一个中文技术助理。",
  "input": [
    {
      "role": "user",
      "content": [
        {
          "type": "input_text",
          "text": "请解释一下 SSE 和 WebSocket 的区别。"
        }
      ]
    }
  ],
  "stream": false,
  "reasoning": {
    "effort": "medium"
  },
  "text": {
    "verbosity": "medium"
  }
}
```

#### 出参说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | `string` | 响应 ID |
| `object` | `string` | 固定为 `response` |
| `model` | `string` | 实际响应模型 |
| `status` | `string` | `completed`、`incomplete` 或 `failed` |
| `output` | `array` | 输出项数组 |
| `usage.input_tokens` | `number` | 输入 token |
| `usage.output_tokens` | `number` | 输出 token |
| `usage.total_tokens` | `number` | 总 token |
| `error` | `object` | 失败时返回 |

#### 出参示例

```json
{
  "id": "resp_abc123",
  "object": "response",
  "model": "gpt-5.4-mini",
  "status": "completed",
  "output": [
    {
      "type": "message",
      "id": "msg_abc123",
      "role": "assistant",
      "status": "completed",
      "content": [
        {
          "type": "output_text",
          "text": "SSE 是服务端单向推送，基于 HTTP 长连接；WebSocket 是全双工连接，适合双向实时交互。SSE 更简单，WebSocket 更灵活。"
        }
      ]
    }
  ],
  "usage": {
    "input_tokens": 36,
    "output_tokens": 41,
    "total_tokens": 77
  }
}
```

---

### `POST /v1/responses/*subpath`

接口名：Responses 子路径

用途：

- 支持 Responses 规范下的扩展子路径。
- 当前代码中已明确保留子路径转发能力，典型示例是 `/v1/responses/compact`。

#### 入参说明

与 `POST /v1/responses` 基本一致。

额外说明：

| 项目 | 说明 |
|------|------|
| 路径参数 | `*subpath` 由客户端自行指定 |
| 请求体 | 与 Responses 协议一致 |
| 适用场景 | 兼容特定 SDK 或上游扩展能力 |

#### 入参示例

```bash
curl -X POST "http://localhost:8080/v1/responses/compact" \
  -H "Authorization: Bearer sk-xxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-5.4-mini",
    "input": "请给出一句简短摘要"
  }'
```

#### 出参示例

```json
{
  "id": "resp_compact_001",
  "object": "response",
  "model": "gpt-5.4-mini",
  "status": "completed",
  "output": [
    {
      "type": "message",
      "role": "assistant",
      "content": [
        {
          "type": "output_text",
          "text": "这是一个简短摘要。"
        }
      ]
    }
  ]
}
```

---

### `GET /v1/responses`

接口名：Responses WebSocket

用途：

- OpenAI Responses 的 WebSocket 入口。
- 需要使用 `Upgrade: websocket` 发起连接。

#### 入参说明

请求头核心字段：

| 字段 | 必填 | 说明 |
|------|------|------|
| `Authorization: Bearer <key>` | 是 | API KEY |
| `Upgrade: websocket` | 是 | WebSocket 升级 |
| `Connection: Upgrade` | 是 | WebSocket 升级 |

连接建立后的**第一条消息**应为 Responses 风格的 JSON，请至少包含：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `model` | `string` | 是 | 模型名 |
| `input` | `string / array` | 否 | 首轮输入 |
| `previous_response_id` | `string` | 否 | 上一轮响应 ID |

#### 入参示例

```json
{
  "model": "gpt-5.4-mini",
  "input": [
    {
      "role": "user",
      "content": [
        {
          "type": "input_text",
          "text": "你好，请开始一个实时对话。"
        }
      ]
    }
  ]
}
```

#### 出参说明

WebSocket 返回的是事件流消息，常见事件类型：

| 字段 | 类型 | 说明 |
|------|------|------|
| `type` | `string` | 事件类型 |
| `response` | `object` | `response.created` / `response.completed` 等事件携带 |
| `item` | `object` | 输出项事件携带 |
| `delta` | `string` | 增量文本 |

#### 出参示例

```json
{
  "type": "response.output_text.delta",
  "output_index": 0,
  "content_index": 0,
  "delta": "你好，"
}
```

---

### `POST /v1/chat/completions`

接口名：OpenAI Chat Completions

用途：

- 使用 OpenAI Chat Completions 协议进行对话。
- 对 Anthropic/OpenAI 不同平台由网关自动适配。

#### 入参说明

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `model` | `string` | 是 | 模型名 |
| `messages` | `array` | 是 | 消息数组 |
| `stream` | `boolean` | 否 | 是否流式 |
| `max_tokens` | `number` | 否 | 最大输出 token |
| `max_completion_tokens` | `number` | 否 | 最大 completion token |
| `temperature` | `number` | 否 | 温度参数 |
| `top_p` | `number` | 否 | top-p 采样 |
| `tools` | `array` | 否 | 工具定义 |
| `tool_choice` | `string / object` | 否 | 工具选择策略 |
| `reasoning_effort` | `string` | 否 | 推理强度 |
| `service_tier` | `string` | 否 | 服务层级 |
| `stop` | `string / array` | 否 | 停止序列 |

`messages[].role` 常见值：

- `system`
- `user`
- `assistant`
- `tool`

#### 入参示例

```json
{
  "model": "gpt-5.4-mini",
  "messages": [
    {
      "role": "system",
      "content": "你是一个严谨的接口文档助手。"
    },
    {
      "role": "user",
      "content": "请解释 POST 和 PUT 的区别。"
    }
  ],
  "stream": false,
  "temperature": 0.2
}
```

#### 出参说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | `string` | 响应 ID |
| `object` | `string` | 一般为 `chat.completion` |
| `created` | `number` | 时间戳 |
| `model` | `string` | 实际响应模型 |
| `choices` | `array` | 候选回复 |
| `choices[].message` | `object` | 助手消息 |
| `choices[].finish_reason` | `string` | 停止原因 |
| `usage` | `object` | token 使用量 |

#### 出参示例

```json
{
  "id": "chatcmpl_abc123",
  "object": "chat.completion",
  "created": 1778544000,
  "model": "gpt-5.4-mini",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "POST 通常用于创建资源，PUT 通常用于整体更新指定资源；PUT 常被设计为幂等，而 POST 不一定幂等。"
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 29,
    "completion_tokens": 32,
    "total_tokens": 61
  }
}
```

---

### `POST /v1/images/generations`

接口名：OpenAI Images Generations

用途：

- 生成图片。
- 仅在 OpenAI 平台分组下可用。

#### 入参说明

请求体核心字段：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `model` | `string` | 否 | 图片模型，未传时项目会补默认值 |
| `prompt` | `string` | 是 | 生图提示词 |
| `n` | `number` | 否 | 生成图片数量，默认 `1` |
| `size` | `string` | 否 | 图片尺寸，如 `1024x1024` |
| `response_format` | `string` | 否 | `url` 或 `b64_json` |
| `quality` | `string` | 否 | 质量级别 |
| `background` | `string` | 否 | 背景策略 |
| `output_format` | `string` | 否 | 输出格式 |
| `stream` | `boolean` | 否 | 是否流式 |

#### 入参示例

```json
{
  "model": "gpt-image-1",
  "prompt": "一张极简风格的云端 API 网关示意图，蓝白配色，扁平化插画",
  "size": "1024x1024",
  "n": 1,
  "response_format": "url"
}
```

#### 出参说明

返回结构兼容 OpenAI Images 风格，常见字段如下：

| 字段 | 类型 | 说明 |
|------|------|------|
| `created` | `number` | 创建时间戳 |
| `data` | `array` | 图片结果数组 |
| `data[].url` | `string` | 图片 URL |
| `data[].b64_json` | `string` | Base64 图片内容 |
| `data[].revised_prompt` | `string` | 修订后的提示词 |

#### 出参示例

```json
{
  "created": 1778544000,
  "data": [
    {
      "url": "https://example.com/generated/image-001.png",
      "revised_prompt": "极简风格的云端 API 网关示意图，蓝白配色，扁平化插画"
    }
  ]
}
```

---

### `POST /v1/images/edits`

接口名：OpenAI Images Edits

用途：

- 基于原图进行图片编辑。
- 支持 JSON 传图片 URL，也支持 `multipart/form-data` 上传图片和蒙版。

#### 入参说明

常见 JSON 字段：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `model` | `string` | 否 | 图片模型 |
| `prompt` | `string` | 否 | 编辑说明 |
| `image` | `string / array` | 否 | 输入图片 URL |
| `mask` | `string` | 否 | 蒙版图片 URL |
| `size` | `string` | 否 | 输出尺寸 |
| `response_format` | `string` | 否 | `url` 或 `b64_json` |
| `quality` | `string` | 否 | 质量级别 |
| `input_fidelity` | `string` | 否 | 输入保真度 |

`multipart/form-data` 常见字段：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `image` | `file` | 是 | 原图 |
| `mask` | `file` | 否 | 蒙版 |
| `prompt` | `string` | 否 | 编辑说明 |
| `model` | `string` | 否 | 模型 |
| `size` | `string` | 否 | 输出尺寸 |

#### 入参示例

```bash
curl -X POST "http://localhost:8080/v1/images/edits" \
  -H "Authorization: Bearer sk-xxxx" \
  -F "image=@input.png" \
  -F "mask=@mask.png" \
  -F "prompt=把背景替换成浅色科技风办公室" \
  -F "model=gpt-image-1"
```

#### 出参说明

与 `/v1/images/generations` 基本一致。

#### 出参示例

```json
{
  "created": 1778544000,
  "data": [
    {
      "url": "https://example.com/edited/image-001.png"
    }
  ]
}
```

---

## Gemini 原生兼容接口

### `GET /v1beta/models`

接口名：Gemini Models List

用途：

- 返回 Gemini 原生 REST 风格模型列表。
- 仅 Gemini 分组或 Antigravity Gemini 兼容模式可用。

#### 入参说明

无请求体，仅需 API KEY。

#### 入参示例

```bash
curl -X GET "http://localhost:8080/v1beta/models" \
  -H "x-goog-api-key: sk-xxxx"
```

#### 出参说明

典型返回字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| `models` | `array` | 模型列表 |
| `models[].name` | `string` | 模型全名，如 `models/gemini-2.5-pro` |
| `models[].displayName` | `string` | 展示名 |
| `models[].description` | `string` | 模型描述 |
| `models[].supportedGenerationMethods` | `array` | 支持的方法 |

#### 出参示例

```json
{
  "models": [
    {
      "name": "models/gemini-2.5-flash",
      "displayName": "Gemini 2.5 Flash",
      "description": "Fast multimodal model",
      "supportedGenerationMethods": [
        "generateContent",
        "streamGenerateContent"
      ]
    }
  ]
}
```

---

### `GET /v1beta/models/:model`

接口名：Gemini Model Detail

用途：

- 获取单个 Gemini 模型的详情。

#### 入参说明

| 参数 | 位置 | 必填 | 说明 |
|------|------|------|------|
| `:model` | path | 是 | 模型名，例如 `gemini-2.5-flash` 或 `models/gemini-2.5-flash` |

#### 入参示例

```bash
curl -X GET "http://localhost:8080/v1beta/models/gemini-2.5-flash" \
  -H "x-goog-api-key: sk-xxxx"
```

#### 出参示例

```json
{
  "name": "models/gemini-2.5-flash",
  "displayName": "Gemini 2.5 Flash",
  "description": "Fast multimodal model",
  "supportedGenerationMethods": [
    "generateContent",
    "streamGenerateContent"
  ]
}
```

---

### `POST /v1beta/models/{model}:generateContent`

接口名：Gemini Generate Content

用途：

- 使用 Gemini 原生 REST 协议发起非流式内容生成。

#### 入参说明

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `contents` | `array` | 是 | 输入内容 |
| `systemInstruction` | `object` | 否 | 系统指令 |
| `generationConfig` | `object` | 否 | 生成参数 |
| `tools` | `array` | 否 | 工具定义 |
| `safetySettings` | `array` | 否 | 安全设置 |

#### 入参示例

```json
{
  "contents": [
    {
      "role": "user",
      "parts": [
        {
          "text": "请用一句话介绍 Gemini native REST API。"
        }
      ]
    }
  ],
  "generationConfig": {
    "temperature": 0.3,
    "maxOutputTokens": 256
  }
}
```

#### 出参说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `candidates` | `array` | 候选结果 |
| `candidates[].content` | `object` | 模型输出内容 |
| `usageMetadata` | `object` | token 使用情况 |
| `modelVersion` | `string` | 模型版本 |

#### 出参示例

```json
{
  "candidates": [
    {
      "content": {
        "role": "model",
        "parts": [
          {
            "text": "Gemini native REST API 是 Google 提供的原生内容生成接口。"
          }
        ]
      },
      "finishReason": "STOP"
    }
  ],
  "usageMetadata": {
    "promptTokenCount": 18,
    "candidatesTokenCount": 17,
    "totalTokenCount": 35
  },
  "modelVersion": "gemini-2.5-flash"
}
```

---

### `POST /v1beta/models/{model}:streamGenerateContent?alt=sse`

接口名：Gemini Stream Generate Content

用途：

- 使用 Gemini 原生 REST 协议进行流式内容生成。

#### 入参说明

入参与 `generateContent` 基本一致。

额外说明：

| 项目 | 说明 |
|------|------|
| 路径动作 | `streamGenerateContent` |
| 查询参数 | 通常携带 `alt=sse` |
| 响应格式 | SSE 事件流 |

#### 入参示例

```bash
curl -N -X POST "http://localhost:8080/v1beta/models/gemini-2.5-flash:streamGenerateContent?alt=sse" \
  -H "x-goog-api-key: sk-xxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "contents": [
      {
        "role": "user",
        "parts": [
          { "text": "请流式输出一句欢迎语。" }
        ]
      }
    ]
  }'
```

#### 出参示例

```text
data: {"candidates":[{"content":{"parts":[{"text":"欢迎"}]}}]}

data: {"candidates":[{"content":{"parts":[{"text":"使用 Sub2API。"}]}}]}
```

---

## Antigravity 专用接口

### `GET /antigravity/models`

接口名：Antigravity 模型列表

用途：

- 返回 Antigravity 平台支持的全部模型列表。

#### 入参示例

```bash
curl -X GET "http://localhost:8080/antigravity/models" \
  -H "Authorization: Bearer sk-xxxx"
```

#### 出参示例

```json
{
  "object": "list",
  "data": [
    {
      "id": "gemini-2.5-flash",
      "type": "model",
      "display_name": "gemini-2.5-flash"
    }
  ]
}
```

---

### `POST /antigravity/v1/messages`

接口名：Antigravity Messages

用途：

- 与 `/v1/messages` 协议一致，但强制使用 Antigravity 账号池，不参与其他平台自动路由。

#### 入参说明

与 `POST /v1/messages` 一致。

#### 入参示例

```json
{
  "model": "gemini-2.5-flash",
  "max_tokens": 512,
  "messages": [
    {
      "role": "user",
      "content": "请介绍一下 Antigravity 路由。"
    }
  ]
}
```

#### 出参示例

```json
{
  "id": "msg_ag_001",
  "type": "message",
  "role": "assistant",
  "content": [
    {
      "type": "text",
      "text": "该接口会固定走 Antigravity 平台账号，不再根据 group.platform 自动分流。"
    }
  ],
  "model": "gemini-2.5-flash",
  "stop_reason": "end_turn",
  "usage": {
    "input_tokens": 16,
    "output_tokens": 23
  }
}
```

---

### `POST /antigravity/v1/messages/count_tokens`

接口名：Antigravity Count Tokens

用途：

- 入口协议与 `/v1/messages/count_tokens` 一致。
- 需要注意：若底层平台不支持 count_tokens，客户端可能收到 `404 not_found_error`，应自行回退到本地估算。

#### 入参说明

与 `POST /v1/messages/count_tokens` 一致。

#### 出参示例

```json
{
  "type": "error",
  "error": {
    "type": "not_found_error",
    "message": "count_tokens endpoint is not supported for this platform"
  }
}
```

---

### `GET /antigravity/v1/models`

接口名：Antigravity v1 模型列表

用途：

- 与 `/antigravity/models` 类似，但路径风格与 `/v1/*` 保持一致。

#### 出参示例

```json
{
  "object": "list",
  "data": [
    {
      "id": "gemini-2.5-flash",
      "type": "model",
      "display_name": "gemini-2.5-flash"
    }
  ]
}
```

---

### `GET /antigravity/v1/usage`

接口名：Antigravity 用量与额度查询

用途：

- 与 `/v1/usage` 返回结构一致，但固定在 Antigravity 路由下使用。

#### 出参示例

```json
{
  "mode": "unrestricted",
  "isValid": true,
  "planName": "钱包余额",
  "remaining": 25.7,
  "unit": "USD",
  "balance": 25.7
}
```

---

### `GET /antigravity/v1beta/models`

接口名：Antigravity Gemini Models List

用途：

- 以 Gemini 原生 REST 风格返回 Antigravity 可用模型。

#### 出参示例

```json
{
  "models": [
    {
      "name": "models/gemini-2.5-flash",
      "displayName": "Gemini 2.5 Flash",
      "supportedGenerationMethods": [
        "generateContent",
        "streamGenerateContent"
      ]
    }
  ]
}
```

---

### `GET /antigravity/v1beta/models/:model`

接口名：Antigravity Gemini Model Detail

用途：

- 获取单个 Antigravity Gemini 风格模型详情。

#### 出参示例

```json
{
  "name": "models/gemini-2.5-flash",
  "displayName": "Gemini 2.5 Flash",
  "supportedGenerationMethods": [
    "generateContent",
    "streamGenerateContent"
  ]
}
```

---

### `POST /antigravity/v1beta/models/{model}:generateContent`

接口名：Antigravity Gemini Generate Content

用途：

- 与 `/v1beta/models/{model}:generateContent` 协议一致，但强制走 Antigravity 平台。

#### 入参示例

```json
{
  "contents": [
    {
      "role": "user",
      "parts": [
        {
          "text": "请用一句话解释 Antigravity Gemini 兼容模式。"
        }
      ]
    }
  ]
}
```

#### 出参示例

```json
{
  "candidates": [
    {
      "content": {
        "role": "model",
        "parts": [
          {
            "text": "该模式允许你用 Gemini 原生 REST 协议访问 Antigravity 账户。"
          }
        ]
      },
      "finishReason": "STOP"
    }
  ]
}
```

---

### `POST /antigravity/v1beta/models/{model}:streamGenerateContent?alt=sse`

接口名：Antigravity Gemini Stream Generate Content

用途：

- 与 `/v1beta/models/{model}:streamGenerateContent?alt=sse` 协议一致，但固定走 Antigravity 平台。

#### 出参示例

```text
data: {"candidates":[{"content":{"parts":[{"text":"你好，"}]}}]}

data: {"candidates":[{"content":{"parts":[{"text":"这里是 Antigravity 流式响应。"}]}}]}
```

---

## 补充说明

### 常见错误响应

通用网关常见错误结构：

```json
{
  "type": "error",
  "error": {
    "type": "invalid_request_error",
    "message": "model is required"
  }
}
```

Gemini 风格接口常见错误结构：

```json
{
  "error": {
    "code": 401,
    "message": "Invalid API key",
    "status": "UNAUTHENTICATED"
  }
}
```

### 建议调用顺序

1. 先调用 `/v1/models` 或 `/v1beta/models` 获取当前 Key 可用模型。
2. 再根据客户端协议选择 `/v1/messages`、`/v1/responses`、`/v1/chat/completions` 或 `/v1beta/models/*`。
3. 需要查询剩余额度或调试状态时调用 `/v1/usage`。
