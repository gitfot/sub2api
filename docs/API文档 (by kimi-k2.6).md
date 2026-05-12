# Sub2API 对外 API 文档

本文档描述所有基于 **API Key** 认证的对外访问接口。

---

## 目录

- [认证方式](#认证方式)
- [Claude API 兼容接口](#claude-api-兼容接口)
  - [POST /v1/messages](#post-v1messages)
  - [POST /v1/messages/count_tokens](#post-v1messagescount_tokens)
  - [GET /v1/models](#get-v1models)
  - [GET /v1/usage](#get-v1usage)
- [OpenAI API 兼容接口](#openai-api-兼容接口)
  - [POST /v1/responses](#post-v1responses)
  - [POST /v1/chat/completions](#post-v1chatcompletions)
  - [POST /v1/images/generations](#post-v1imagesgenerations)
  - [POST /v1/images/edits](#post-v1imagesedits)
  - [GET /v1/responses (WebSocket)](#get-v1responses-websocket)
- [Gemini API 兼容接口](#gemini-api-兼容接口)
  - [GET /v1beta/models](#get-v1betamodels)
  - [GET /v1beta/models/{model}](#get-v1betamodelsmodel)
  - [POST /v1beta/models/{model}:{action}](#post-v1betamodelsmodelaction)
- [别名路由](#别名路由)
- [Antigravity 专用路由](#antigravity-专用路由)
- [通用错误响应](#通用错误响应)

---

## 认证方式

所有接口均需在请求中携带 API Key，支持以下三种传递方式（按优先级）：

| 方式 | Header 示例 |
|------|------------|
| Authorization Bearer | `Authorization: Bearer sk-your-api-key` |
| x-api-key | `x-api-key: sk-your-api-key` |
| x-goog-api-key (Gemini CLI 兼容) | `x-goog-api-key: sk-your-api-key` |

> 废弃方式：Query 参数 `?key=` 或 `?api_key=` 仍支持但不推荐使用。

---

## Claude API 兼容接口

### POST /v1/messages

Claude Messages API 兼容接口。根据 API Key 所属分组的平台配置，自动路由到 Anthropic 或 OpenAI 上游。

| 属性 | 值 |
|------|-----|
| 方法 | POST |
| Content-Type | application/json |
| 流式支持 | 是（`stream: true` 时返回 SSE） |

#### 入参

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| model | string | 是 | 模型名称，如 `claude-3-7-sonnet-20250219` |
| messages | array | 是 | 对话消息数组，每项包含 `role` 和 `content` |
| max_tokens | integer | 是 | 最大生成 token 数 |
| stream | boolean | 否 | 是否流式返回，默认 `false` |
| system | string/object/array | 否 | 系统提示词 |
| thinking | object | 否 | 思考模式配置，如 `{"type": "enabled", "budget_tokens": 32000}` |
| temperature | number | 否 | 采样温度，默认 `1.0` |
| top_p | number | 否 | 核采样概率 |
| top_k | integer | 否 | Top-K 采样 |
| metadata | object | 否 | 元数据，可包含 `user_id` 用于会话粘性 |
| tools | array | 否 | 工具定义数组 |
| tool_choice | object/string | 否 | 工具选择策略 |
| stop_sequences | array | 否 | 停止序列数组 |

#### 入参示例（非流式）

```json
{
  "model": "claude-3-7-sonnet-20250219",
  "messages": [
    {
      "role": "user",
      "content": "你好，请介绍一下自己"
    }
  ],
  "max_tokens": 1024,
  "temperature": 0.7,
  "metadata": {
    "user_id": "user-12345"
  }
}
```

#### 入参示例（流式）

```json
{
  "model": "claude-3-7-sonnet-20250219",
  "messages": [
    {
      "role": "user",
      "content": "你好"
    }
  ],
  "max_tokens": 1024,
  "stream": true
}
```

#### 出参（非流式）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 消息唯一标识 |
| type | string | 固定为 `"message"` |
| role | string | 固定为 `"assistant"` |
| model | string | 实际使用的模型 |
| content | array | 内容块数组，每项含 `type` 和对应字段 |
| stop_reason | string/null | 停止原因，如 `"end_turn"`、`"max_tokens"` |
| stop_sequence | string/null | 触发的停止序列 |
| usage | object | token 使用量统计 |

#### 出参示例（非流式）

```json
{
  "id": "msg_bdrk_xxxxxxxxxxxxxxxxxxxxxxxx",
  "type": "message",
  "role": "assistant",
  "model": "claude-3-7-sonnet-20250219",
  "content": [
    {
      "type": "text",
      "text": "你好！我是 Claude，一个 AI 助手。"
    }
  ],
  "stop_reason": "end_turn",
  "stop_sequence": null,
  "usage": {
    "input_tokens": 15,
    "output_tokens": 25,
    "cache_creation_input_tokens": 0,
    "cache_read_input_tokens": 0,
    "total_tokens": 40
  }
}
```

#### 出参（流式 SSE）

流式响应为 `text/event-stream` 格式，事件类型包括：

| 事件 | 说明 |
|------|------|
| `message_start` | 消息开始 |
| `content_block_start` | 内容块开始 |
| `content_block_delta` | 内容块增量（文本片段） |
| `content_block_stop` | 内容块结束 |
| `message_delta` | 消息增量（含 usage） |
| `message_stop` | 消息结束 |

#### SSE 示例

```
event: message_start
data: {"type":"message_start","message":{"id":"msg_xxx","type":"message","role":"assistant","model":"claude-3-7-sonnet","content":[],"stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":10,"output_tokens":0}}}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"你好"}}

event: message_stop
data: {"type":"message_stop"}
```

---

### POST /v1/messages/count_tokens

Token 计数接口。校验计费资格但不计入并发、不记录使用量。

> 注意：OpenAI 平台分组调用此接口会返回 404。

| 属性 | 值 |
|------|-----|
| 方法 | POST |
| Content-Type | application/json |

#### 入参

与 `/v1/messages` 相同，但 `stream` 无效。

#### 入参示例

```json
{
  "model": "claude-3-7-sonnet-20250219",
  "messages": [
    {
      "role": "user",
      "content": "你好"
    }
  ]
}
```

#### 出参

| 字段 | 类型 | 说明 |
|------|------|------|
| input_tokens | integer | 输入 token 数 |

#### 出参示例

```json
{
  "input_tokens": 15
}
```

---

### GET /v1/models

获取当前分组可用的模型列表。

| 属性 | 值 |
|------|-----|
| 方法 | GET |

#### 入参

无。

#### 出参

| 字段 | 类型 | 说明 |
|------|------|------|
| object | string | 固定为 `"list"` |
| data | array | 模型列表，每项含 `id`、`type`、`display_name`、`created_at` |

#### 出参示例

```json
{
  "object": "list",
  "data": [
    {
      "id": "claude-3-7-sonnet-20250219",
      "type": "model",
      "display_name": "claude-3-7-sonnet-20250219",
      "created_at": "2024-01-01T00:00:00Z"
    },
    {
      "id": "claude-3-5-sonnet-20241022",
      "type": "model",
      "display_name": "claude-3-5-sonnet-20241022",
      "created_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

---

### GET /v1/usage

查询 API Key 的余额、配额和使用统计。此接口仅做认证，不校验计费。

| 属性 | 值 |
|------|-----|
| 方法 | GET |

#### Query 参数

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| start_date | string | 否 | 统计开始日期，格式 `YYYY-MM-DD`，默认近 30 天 |
| end_date | string | 否 | 统计结束日期，格式 `YYYY-MM-DD` |

#### 出参

响应分为两种模式：`quota_limited`（API Key 有额度/速率限制）和 `unrestricted`（无限制，返回订阅或钱包信息）。

**quota_limited 模式：**

| 字段 | 类型 | 说明 |
|------|------|------|
| mode | string | `"quota_limited"` |
| isValid | boolean | API Key 是否有效 |
| status | string | Key 状态：`active`、`expired`、`quota_exhausted`、`disabled` |
| quota | object | 额度信息（limit/used/remaining/unit） |
| rate_limits | array | 速率限制详情（5h/1d/7d 窗口） |
| expires_at | string (ISO8601) | 过期时间 |
| days_until_expiry | integer | 距离过期天数 |
| usage | object | 今日/总计用量统计 |
| model_stats | array | 按模型统计的用量 |

**unrestricted 模式：**

| 字段 | 类型 | 说明 |
|------|------|------|
| mode | string | `"unrestricted"` |
| isValid | boolean | 是否有效 |
| planName | string | 计划名称或 `"钱包余额"` |
| remaining | number | 剩余额度（USD） |
| unit | string | 固定为 `"USD"` |
| balance | number | 钱包余额（余额模式） |
| subscription | object | 订阅信息（订阅模式） |
| usage | object | 用量统计 |
| model_stats | array | 按模型统计 |

#### 出参示例（quota_limited）

```json
{
  "mode": "quota_limited",
  "isValid": true,
  "status": "active",
  "quota": {
    "limit": 50.00,
    "used": 12.50,
    "remaining": 37.50,
    "unit": "USD"
  },
  "rate_limits": [
    {
      "window": "1d",
      "limit": 1000,
      "used": 120,
      "remaining": 880,
      "window_start": "2026-05-12T00:00:00Z",
      "reset_at": "2026-05-13T00:00:00Z"
    }
  ],
  "usage": {
    "today": {
      "requests": 45,
      "input_tokens": 12000,
      "output_tokens": 8000,
      "total_tokens": 20000,
      "cost": 0.50,
      "actual_cost": 0.48
    },
    "total": {
      "requests": 1200,
      "input_tokens": 350000,
      "output_tokens": 180000,
      "total_tokens": 530000,
      "cost": 12.50,
      "actual_cost": 12.10
    }
  }
}
```

#### 出参示例（unrestricted - 钱包模式）

```json
{
  "mode": "unrestricted",
  "isValid": true,
  "planName": "钱包余额",
  "remaining": 156.78,
  "unit": "USD",
  "balance": 156.78,
  "usage": {
    "today": {
      "requests": 45,
      "input_tokens": 12000,
      "output_tokens": 8000,
      "total_tokens": 20000,
      "cost": 0.50
    }
  }
}
```

---

## OpenAI API 兼容接口

### POST /v1/responses

OpenAI Responses API 兼容接口。支持流式（SSE）和非流式响应。

| 属性 | 值 |
|------|-----|
| 方法 | POST |
| Content-Type | application/json |
| 流式支持 | 是 |

#### 入参

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| model | string | 是 | 模型名称，如 `gpt-4o` |
| input | string/array | 是 | 用户输入（文本或消息数组） |
| stream | boolean | 否 | 是否流式返回 |
| previous_response_id | string | 否 | 前一轮响应 ID（仅 WebSocket v2 支持） |
| instructions | string | 否 | 系统指令 |
| temperature | number | 否 | 采样温度 |
| top_p | number | 否 | 核采样概率 |
| max_output_tokens | integer | 否 | 最大输出 token 数 |
| tools | array | 否 | 工具定义 |
| tool_choice | string/object | 否 | 工具选择策略 |
| text | object | 否 | 文本输出格式配置 |
| reasoning | object | 否 | 推理参数配置 |
| store | boolean | 否 | 是否存储响应 |

#### 入参示例（非流式）

```json
{
  "model": "gpt-4o",
  "input": "你好，请介绍一下自己",
  "temperature": 0.7,
  "max_output_tokens": 1024
}
```

#### 入参示例（流式）

```json
{
  "model": "gpt-4o",
  "input": "你好",
  "stream": true
}
```

#### 出参（非流式）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 响应唯一标识（`resp_` 前缀） |
| object | string | 固定为 `"response"` |
| status | string | 响应状态：`completed`、`in_progress` 等 |
| model | string | 实际使用的模型 |
| output | array | 输出项数组（消息、工具调用等） |
| usage | object | token 使用量 |
| created_at | integer | 创建时间戳（Unix 秒） |

#### 出参示例（非流式）

```json
{
  "id": "resp_xxxxxxxxxxxxxxxxxxxxxxxx",
  "object": "response",
  "status": "completed",
  "model": "gpt-4o",
  "output": [
    {
      "type": "message",
      "role": "assistant",
      "content": [
        {
          "type": "output_text",
          "text": "你好！我是 GPT-4o，一个 AI 助手。",
          "annotations": []
        }
      ]
    }
  ],
  "usage": {
    "input_tokens": 15,
    "output_tokens": 20,
    "total_tokens": 35
  },
  "created_at": 1715500000
}
```

#### 出参（流式 SSE）

流式响应事件类型：

| 事件 | 说明 |
|------|------|
| `response.created` | 响应创建 |
| `response.in_progress` | 响应处理中 |
| `response.output_item.added` | 输出项添加 |
| `response.content_part.added` | 内容部分添加 |
| `response.output_text.delta` | 文本增量 |
| `response.content_part.done` | 内容部分完成 |
| `response.output_item.done` | 输出项完成 |
| `response.completed` | 响应完成 |

---

### POST /v1/chat/completions

OpenAI Chat Completions API 兼容接口。

| 属性 | 值 |
|------|-----|
| 方法 | POST |
| Content-Type | application/json |
| 流式支持 | 是 |

#### 入参

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| model | string | 是 | 模型名称 |
| messages | array | 是 | 对话消息数组，每项含 `role` 和 `content` |
| stream | boolean | 否 | 是否流式返回 |
| temperature | number | 否 | 采样温度 |
| top_p | number | 否 | 核采样概率 |
| max_tokens | integer | 否 | 最大生成 token 数 |
| tools | array | 否 | 工具定义 |
| tool_choice | string/object | 否 | 工具选择策略 |
| response_format | object | 否 | 响应格式约束 |
| stop | string/array | 否 | 停止序列 |

#### 入参示例

```json
{
  "model": "gpt-4o",
  "messages": [
    {
      "role": "system",
      "content": "你是一个有用的助手"
    },
    {
      "role": "user",
      "content": "你好"
    }
  ],
  "temperature": 0.7,
  "max_tokens": 1024
}
```

#### 出参

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 响应 ID |
| object | string | 固定为 `"chat.completion"` |
| model | string | 实际使用的模型 |
| choices | array | 生成结果数组 |
| usage | object | token 使用量 |
| created | integer | 创建时间戳 |

#### 出参示例

```json
{
  "id": "chatcmpl-xxxxxxxxxxxxxxxx",
  "object": "chat.completion",
  "model": "gpt-4o",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "你好！有什么我可以帮助你的吗？"
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 25,
    "completion_tokens": 15,
    "total_tokens": 40
  },
  "created": 1715500000
}
```

---

### POST /v1/images/generations

OpenAI 图片生成接口（仅限 OpenAI 平台分组）。

| 属性 | 值 |
|------|-----|
| 方法 | POST |
| Content-Type | application/json |

#### 入参

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| model | string | 是 | 图像模型，如 `dall-e-3` |
| prompt | string | 是 | 图像描述提示词 |
| n | integer | 否 | 生成数量，默认 `1` |
| size | string | 否 | 图像尺寸，如 `1024x1024` |
| quality | string | 否 | 图像质量：`standard`、`hd` |
| style | string | 否 | 图像风格：`vivid`、`natural` |
| response_format | string | 否 | 响应格式：`url`、`b64_json` |

#### 入参示例

```json
{
  "model": "dall-e-3",
  "prompt": "一只可爱的橘猫在草地上玩耍",
  "n": 1,
  "size": "1024x1024",
  "quality": "standard"
}
```

#### 出参

| 字段 | 类型 | 说明 |
|------|------|------|
| created | integer | 创建时间戳 |
| data | array | 生成结果数组，每项含 `url` 或 `b64_json` |

#### 出参示例

```json
{
  "created": 1715500000,
  "data": [
    {
      "url": "https://example.com/generated-image.png",
      "revised_prompt": "A cute orange cat playing on a grassy field"
    }
  ]
}
```

---

### POST /v1/images/edits

OpenAI 图片编辑接口（仅限 OpenAI 平台分组）。

| 属性 | 值 |
|------|-----|
| 方法 | POST |
| Content-Type | multipart/form-data |

#### 入参

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| image | file | 是 | 待编辑的原始图片 |
| prompt | string | 是 | 编辑指令 |
| mask | file | 否 | 蒙版图片 |
| model | string | 否 | 图像模型 |
| n | integer | 否 | 生成数量 |
| size | string | 否 | 图像尺寸 |
| response_format | string | 否 | 响应格式 |

#### 出参

与 `/v1/images/generations` 相同。

---

### GET /v1/responses (WebSocket)

OpenAI Responses API WebSocket v2 入口。支持多轮对话保持长连接。

| 属性 | 值 |
|------|-----|
| 方法 | GET |
| 协议升级 | WebSocket (`Upgrade: websocket`) |

#### 连接流程

1. 客户端发送 WebSocket 握手请求到 `GET /v1/responses`
2. 服务端接受连接后，等待客户端发送首条 `response.create` JSON 消息
3. 首条消息格式与 POST `/v1/responses` 请求体相同
4. 后续可在同一连接中继续发送新的请求消息

#### 首条消息示例

```json
{
  "model": "gpt-4o",
  "input": "你好",
  "previous_response_id": "resp_xxxxxxxxxxxxxxxxxxxxxxxx"
}
```

#### 响应格式

WebSocket 返回与 SSE 流式相同的事件 JSON，但直接以 WebSocket 文本消息传输。

---

## Gemini API 兼容接口

### GET /v1beta/models

获取 Gemini 模型列表。

| 属性 | 值 |
|------|-----|
| 方法 | GET |

#### 出参

Google API 标准格式模型列表。

#### 出参示例

```json
{
  "models": [
    {
      "name": "models/gemini-2.5-pro-preview-03-25",
      "version": "2.5",
      "displayName": "Gemini 2.5 Pro Preview",
      "description": "Gemini 2.5 Pro 预览版",
      "inputTokenLimit": 1048576,
      "outputTokenLimit": 8192,
      "supportedGenerationMethods": ["generateContent", "countTokens"]
    }
  ]
}
```

---

### GET /v1beta/models/{model}

获取指定 Gemini 模型信息。

| 属性 | 值 |
|------|-----|
| 方法 | GET |
| 路径参数 | `model` - 模型名称，如 `gemini-2.5-pro-preview-03-25` |

#### 出参示例

```json
{
  "name": "models/gemini-2.5-pro-preview-03-25",
  "version": "2.5",
  "displayName": "Gemini 2.5 Pro Preview",
  "description": "Gemini 2.5 Pro 预览版",
  "inputTokenLimit": 1048576,
  "outputTokenLimit": 8192,
  "supportedGenerationMethods": ["generateContent", "countTokens"]
}
```

---

### POST /v1beta/models/{model}:{action}

Gemini 原生内容生成接口。

| 属性 | 值 |
|------|-----|
| 方法 | POST |
| Content-Type | application/json |
| 路径参数 | `model` - 模型名称；`action` - 操作类型 |

#### 支持的操作

| action | 说明 | 流式 |
|--------|------|------|
| `generateContent` | 内容生成 | 否 |
| `streamGenerateContent` | 流式内容生成 | 是（SSE） |

#### 入参

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| contents | array | 是 | 对话内容数组，每项含 `role` 和 `parts` |
| systemInstruction | object | 否 | 系统指令，含 `parts` 数组 |
| generationConfig | object | 否 | 生成配置（temperature、topP、maxOutputTokens 等） |
| safetySettings | array | 否 | 安全设置数组 |
| tools | array | 否 | 工具定义 |

#### 入参示例

```json
{
  "contents": [
    {
      "role": "user",
      "parts": [
        {
          "text": "你好，请介绍一下自己"
        }
      ]
    }
  ],
  "generationConfig": {
    "temperature": 0.7,
    "maxOutputTokens": 1024
  }
}
```

#### 出参（generateContent）

| 字段 | 类型 | 说明 |
|------|------|------|
| candidates | array | 候选结果数组 |
| usageMetadata | object | token 使用量 |

#### 出参示例

```json
{
  "candidates": [
    {
      "content": {
        "role": "model",
        "parts": [
          {
            "text": "你好！我是 Gemini，一个 AI 助手。"
          }
        ]
      },
      "finishReason": "STOP"
    }
  ],
  "usageMetadata": {
    "promptTokenCount": 10,
    "candidatesTokenCount": 20,
    "totalTokenCount": 30
  }
}
```

---

## 别名路由

以下路由与带 `/v1` 前缀的接口功能完全相同，用于兼容不同客户端：

| 方法 | 路径 | 等价于 |
|------|------|--------|
| POST | `/responses` | POST `/v1/responses` |
| POST | `/responses/*subpath` | POST `/v1/responses/*subpath` |
| GET | `/responses` | GET `/v1/responses` (WebSocket) |
| POST | `/chat/completions` | POST `/v1/chat/completions` |
| POST | `/images/generations` | POST `/v1/images/generations` |
| POST | `/images/edits` | POST `/v1/images/edits` |
| GET | `/antigravity/models` | 返回 Antigravity 模型列表 |
| GET | `/backend-api/codex/responses` | Codex CLI 兼容 Responses |
| GET | `/backend-api/codex/responses/*subpath` | Codex CLI 兼容 Responses 子路径 |

---

## Antigravity 专用路由

以下路由强制使用 Antigravity 平台账号，不混合调度其他平台：

### POST /antigravity/v1/messages

与 `/v1/messages` 参数和响应相同，但强制路由到 Antigravity 账号。

### POST /antigravity/v1/messages/count_tokens

与 `/v1/messages/count_tokens` 相同，强制 Antigravity 平台。

### GET /antigravity/v1/models

返回 Antigravity 平台支持的模型列表。

### GET /antigravity/v1/usage

与 `/v1/usage` 相同。

### GET /antigravity/v1beta/models

与 `/v1beta/models` 相同，强制 Antigravity 平台。

### GET /antigravity/v1beta/models/{model}

与 `/v1beta/models/{model}` 相同，强制 Antigravity 平台。

### POST /antigravity/v1beta/models/{model}:{action}

与 `/v1beta/models/{model}:{action}` 相同，强制 Antigravity 平台。

---

## 通用错误响应

### Claude 格式错误（/v1/* 路由，除 OpenAI 专属接口外）

```json
{
  "type": "error",
  "error": {
    "type": "authentication_error",
    "message": "Invalid API key"
  }
}
```

### OpenAI 格式错误（/v1/responses、/v1/chat/completions 等）

```json
{
  "error": {
    "type": "authentication_error",
    "message": "Invalid API key"
  }
}
```

### Gemini/Google 格式错误（/v1beta/* 路由）

```json
{
  "error": {
    "code": 401,
    "message": "Invalid API key",
    "status": "UNAUTHENTICATED"
  }
}
```

### 常见错误码

| HTTP 状态码 | error type | 说明 |
|-------------|-----------|------|
| 400 | `invalid_request_error` | 请求参数错误 |
| 401 | `authentication_error` | API Key 无效或未提供 |
| 403 | `permission_error` / `billing_error` | 权限不足或余额不足 |
| 429 | `rate_limit_error` | 速率限制或并发超限 |
| 502 | `upstream_error` | 上游服务错误 |
| 503 | `api_error` / `overloaded_error` | 无可用账号或上游过载 |
| 529 | `overloaded_error` | 上游服务过载 |
