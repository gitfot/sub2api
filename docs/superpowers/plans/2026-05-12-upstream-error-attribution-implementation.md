# Upstream Error Attribution Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make upstream 4xx/5xx responses count as upstream errors while preserving client-visible request-error counting, and show the exact upstream account in Ops error details.

**Architecture:** Keep the existing `ops_error_logs` schema and reuse its upstream context fields. Backend error logging will derive provider attribution from `OpsUpstreamStatusCodeKey` / `OpsUpstreamErrorsKey` before persisting, while frontend details will parse inline `upstream_errors` first and only use correlated upstream logs as a supplement.

**Tech Stack:** Go, Gin, PostgreSQL-backed Ops repository, Vue 3, TypeScript, Vitest.

---

## File Structure

**Create**

- `frontend/src/views/admin/ops/utils/errorDetailUpstream.ts`: converts `OpsErrorDetail.upstream_errors` and single upstream fields into display-ready upstream attempts.
- `frontend/src/views/admin/ops/utils/__tests__/errorDetailUpstream.spec.ts`: unit coverage for upstream attempt parsing and fallback synthesis.
- `frontend/src/views/admin/ops/components/__tests__/OpsErrorDetailModal.spec.ts`: component coverage that request-error details show both user and upstream account context.
- `backend/internal/repository/ops_repo_dashboard_test.go`: repository-level regression coverage for provider 400 upstream dashboard counts.

**Modify**

- `backend/internal/handler/ops_error_logger.go`: add upstream-attribution helpers and apply them when building `OpsInsertErrorLogInput`.
- `backend/internal/handler/ops_error_logger_test.go`: unit coverage for the new attribution helpers.
- `frontend/src/views/admin/ops/components/OpsErrorDetailModal.vue`: show separate user and upstream-account cards, and render inline upstream attempts.
- `frontend/src/views/admin/ops/utils/errorDetailResponse.ts`: keep primary response body behavior unchanged, import nothing from display parsing.

## Data And Behavior Notes

- `status_code` remains the client-facing status code and continues to drive request-error counts.
- `error_type` remains the client-facing error type and can stay `invalid_request_error`.
- `upstream_status_code >= 400` or any `upstream_errors[*].upstream_status_code >= 400` means provider attribution.
- Provider attribution sets `error_phase = upstream`, `error_owner = provider`, and `error_source = upstream_http`.
- Local validation failures such as missing `model` have no upstream context and stay `request/client`.
- Account success-rate aggregation remains based on `status_code >= 400 AND account_id IS NOT NULL`; it must not filter by `error_owner`.

### Task 1: Lock Backend Attribution Semantics

**Files:**

- Modify: `backend/internal/handler/ops_error_logger_test.go`

- [ ] **Step 1: Add failing helper tests**

Append these tests to `backend/internal/handler/ops_error_logger_test.go`:

```go
func TestOpsHasProviderUpstreamErrorContext_StatusCode400(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set(service.OpsUpstreamStatusCodeKey, 400)

	require.True(t, opsHasProviderUpstreamErrorContext(c))
}

func TestOpsHasProviderUpstreamErrorContext_EventStatusCode400(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set(service.OpsUpstreamErrorsKey, []*service.OpsUpstreamErrorEvent{
		{AccountID: 101, UpstreamStatusCode: 400, Message: "openai_error"},
	})

	require.True(t, opsHasProviderUpstreamErrorContext(c))
}

func TestOpsHasProviderUpstreamErrorContext_NoUpstreamError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	require.False(t, opsHasProviderUpstreamErrorContext(c))

	c.Set(service.OpsUpstreamStatusCodeKey, 200)
	require.False(t, opsHasProviderUpstreamErrorContext(c))

	c.Set(service.OpsUpstreamErrorsKey, []*service.OpsUpstreamErrorEvent{
		{AccountID: 101, UpstreamStatusCode: 0, Message: "dial timeout"},
	})
	require.False(t, opsHasProviderUpstreamErrorContext(c))
}

func TestOpsApplyProviderAttributionIfUpstreamError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set(service.OpsUpstreamStatusCodeKey, int64(422))

	phase, owner, source := opsApplyProviderAttributionIfUpstreamError(c, "request", "client", "client_request")

	require.Equal(t, "upstream", phase)
	require.Equal(t, "provider", owner)
	require.Equal(t, "upstream_http", source)
}

func TestOpsApplyProviderAttributionIfUpstreamError_KeepsClientAttributionWithoutUpstream(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	phase, owner, source := opsApplyProviderAttributionIfUpstreamError(c, "request", "client", "client_request")

	require.Equal(t, "request", phase)
	require.Equal(t, "client", owner)
	require.Equal(t, "client_request", source)
}
```

- [ ] **Step 2: Run the targeted backend test and verify it fails**

Run:

```bash
cd /mnt/d/wanwan/project/self/APIStation/sub2api/backend
go test ./internal/handler -run 'TestOpsHasProviderUpstreamErrorContext|TestOpsApplyProviderAttributionIfUpstreamError' -count=1
```

Expected: FAIL with undefined `opsHasProviderUpstreamErrorContext` and `opsApplyProviderAttributionIfUpstreamError`.

- [ ] **Step 3: Commit the failing tests**

```bash
git add -- backend/internal/handler/ops_error_logger_test.go
git commit -m "test: cover upstream error attribution helpers"
```

### Task 2: Implement Backend Provider Attribution

**Files:**

- Modify: `backend/internal/handler/ops_error_logger.go`
- Modify: `backend/internal/handler/ops_error_logger_test.go`

- [ ] **Step 1: Add attribution helper functions**

Add these helpers near the existing classification helpers in `backend/internal/handler/ops_error_logger.go`, before `parseOpsErrorResponse`:

```go
func opsHasProviderUpstreamErrorContext(c *gin.Context) bool {
	if c == nil {
		return false
	}
	if v, ok := c.Get(service.OpsUpstreamStatusCodeKey); ok {
		switch t := v.(type) {
		case int:
			if t >= 400 {
				return true
			}
		case int64:
			if t >= 400 {
				return true
			}
		}
	}
	if v, ok := c.Get(service.OpsUpstreamErrorsKey); ok {
		if events, ok := v.([]*service.OpsUpstreamErrorEvent); ok {
			for _, ev := range events {
				if ev != nil && ev.UpstreamStatusCode >= 400 {
					return true
				}
			}
		}
	}
	return false
}

func opsApplyProviderAttributionIfUpstreamError(c *gin.Context, phase, owner, source string) (string, string, string) {
	if !opsHasProviderUpstreamErrorContext(c) {
		return phase, owner, source
	}
	return "upstream", "provider", "upstream_http"
}
```

- [ ] **Step 2: Apply attribution before building the error entry**

In the `status >= 400` branch of `OpsErrorLoggerMiddleware`, immediately after existing classification:

```go
phase := classifyOpsPhase(normalizedType, parsed.Message, parsed.Code)
isBusinessLimited := classifyOpsIsBusinessLimited(normalizedType, phase, parsed.Code, status, parsed.Message)

errorOwner := classifyOpsErrorOwner(phase, parsed.Message)
errorSource := classifyOpsErrorSource(phase, parsed.Message)
phase, errorOwner, errorSource = opsApplyProviderAttributionIfUpstreamError(c, phase, errorOwner, errorSource)
```

Keep the existing upstream-field capture block unchanged so `UpstreamStatusCode`, `UpstreamErrorMessage`, `UpstreamErrorDetail`, and `UpstreamErrors` are still copied into the entry.

- [ ] **Step 3: Run the new helper tests**

Run:

```bash
cd /mnt/d/wanwan/project/self/APIStation/sub2api/backend
go test ./internal/handler -run 'TestOpsHasProviderUpstreamErrorContext|TestOpsApplyProviderAttributionIfUpstreamError' -count=1
```

Expected: PASS.

- [ ] **Step 4: Run the full handler package tests**

Run:

```bash
cd /mnt/d/wanwan/project/self/APIStation/sub2api/backend
go test ./internal/handler -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit backend attribution implementation**

```bash
git add -- backend/internal/handler/ops_error_logger.go backend/internal/handler/ops_error_logger_test.go
git commit -m "fix: attribute upstream http errors to provider"
```

### Task 3: Add Backend Statistics Regression Coverage

**Files:**

- Create: `backend/internal/repository/ops_repo_dashboard_test.go`

- [ ] **Step 1: Add repository tests for provider 400 and client 400 dashboard rows**

Create `backend/internal/repository/ops_repo_dashboard_test.go`:

```go
package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestOpsRepositoryQueryErrorCounts_Provider400CountsAsUpstream(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &opsRepository{db: db}

	start := time.Date(2026, 5, 12, 10, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)

	mock.ExpectQuery(`(?s)error_owner = 'provider'.+upstream_excl`).
		WithArgs(start, end).
		WillReturnRows(sqlmock.NewRows([]string{
			"error_total",
			"business_limited",
			"error_sla",
			"upstream_excl",
			"upstream_429",
			"upstream_529",
		}).AddRow(int64(1), int64(0), int64(1), int64(1), int64(0), int64(0)))

	total, businessLimited, sla, upstreamExcl, upstream429, upstream529, err := repo.queryErrorCounts(
		context.Background(),
		&service.OpsDashboardFilter{},
		start,
		end,
	)

	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Equal(t, int64(0), businessLimited)
	require.Equal(t, int64(1), sla)
	require.Equal(t, int64(1), upstreamExcl)
	require.Equal(t, int64(0), upstream429)
	require.Equal(t, int64(0), upstream529)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOpsRepositoryQueryErrorCounts_Client400DoesNotCountAsUpstream(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &opsRepository{db: db}

	start := time.Date(2026, 5, 12, 11, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)

	mock.ExpectQuery(`(?s)error_owner = 'provider'.+upstream_excl`).
		WithArgs(start, end).
		WillReturnRows(sqlmock.NewRows([]string{
			"error_total",
			"business_limited",
			"error_sla",
			"upstream_excl",
			"upstream_429",
			"upstream_529",
		}).AddRow(int64(1), int64(0), int64(1), int64(0), int64(0), int64(0)))

	total, _, sla, upstreamExcl, upstream429, upstream529, err := repo.queryErrorCounts(
		context.Background(),
		&service.OpsDashboardFilter{},
		start,
		end,
	)

	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Equal(t, int64(1), sla)
	require.Equal(t, int64(0), upstreamExcl)
	require.Equal(t, int64(0), upstream429)
	require.Equal(t, int64(0), upstream529)
	require.NoError(t, mock.ExpectationsWereMet())
}
```

- [ ] **Step 2: Run repository tests**

Run:

```bash
cd /mnt/d/wanwan/project/self/APIStation/sub2api/backend
go test ./internal/repository -run 'TestOpsRepositoryQueryErrorCounts_' -count=1
```

Expected: PASS. These tests lock the dashboard query contract; Task 2 is what makes real upstream 400 rows receive `error_owner = provider`.

- [ ] **Step 3: Run account success-rate aggregation tests**

Run:

```bash
cd /mnt/d/wanwan/project/self/APIStation/sub2api/backend
go test ./internal/repository -run 'Test.*AccountRequestStats|Test.*SuccessRate' -count=1
```

Expected: PASS. Existing account success-rate tests should remain unchanged because success-rate failure counting is still based on `status_code >= 400`, not `error_owner`.

- [ ] **Step 4: Commit statistics regression tests**

```bash
git add -- backend/internal/repository/ops_repo_dashboard_test.go
git commit -m "test: cover provider upstream error dashboard counts"
```

### Task 4: Add Frontend Upstream Attempt Parsing

**Files:**

- Create: `frontend/src/views/admin/ops/utils/errorDetailUpstream.ts`
- Create: `frontend/src/views/admin/ops/utils/__tests__/errorDetailUpstream.spec.ts`

- [ ] **Step 1: Write failing utility tests**

Create `frontend/src/views/admin/ops/utils/__tests__/errorDetailUpstream.spec.ts`:

```ts
import { describe, expect, it } from 'vitest'
import type { OpsErrorDetail } from '@/api/admin/ops'
import { hasUpstreamAttribution, resolveDisplayUpstreamEvents } from '../errorDetailUpstream'

function makeDetail(overrides: Partial<OpsErrorDetail>): OpsErrorDetail {
  return {
    id: 1,
    created_at: '2026-05-12T10:00:00Z',
    phase: 'request',
    type: 'invalid_request_error',
    error_owner: 'provider',
    error_source: 'upstream_http',
    severity: 'P3',
    status_code: 400,
    platform: 'openai',
    model: 'gpt-4o-mini',
    is_retryable: false,
    retry_count: 0,
    resolved: false,
    client_request_id: 'client-rid-1',
    request_id: 'gw-rid-1',
    message: 'openai_error',
    user_email: 'user@example.com',
    account_id: 101,
    account_name: 'openai-upstream-01',
    group_name: 'default',
    error_body: '{"error":{"message":"openai_error","type":"invalid_request_error"}}',
    user_agent: '',
    request_body: '',
    request_body_truncated: false,
    is_business_limited: false,
    ...overrides
  }
}

describe('errorDetailUpstream', () => {
  it('parses inline upstream_errors events into display rows', () => {
    const detail = makeDetail({
      upstream_errors: JSON.stringify([
        {
          account_id: 101,
          account_name: 'openai-upstream-01',
          platform: 'openai',
          upstream_status_code: 400,
          upstream_request_id: 'req-upstream-1',
          kind: 'http_error',
          message: 'openai_error',
          detail: '{"error":{"message":"openai_error"}}',
          upstream_response_body: '{"error":{"message":"openai_error"}}'
        }
      ])
    })

    const events = resolveDisplayUpstreamEvents(detail)

    expect(events).toHaveLength(1)
    expect(events[0]).toMatchObject({
      key: 'inline-0',
      account_id: 101,
      account_name: 'openai-upstream-01',
      platform: 'openai',
      status_code: 400,
      request_id: 'req-upstream-1',
      kind: 'http_error',
      message: 'openai_error'
    })
    expect(events[0].response_preview).toContain('openai_error')
  })

  it('builds a fallback event from single upstream fields', () => {
    const detail = makeDetail({
      upstream_errors: '',
      upstream_status_code: 422,
      upstream_error_message: 'schema rejected',
      upstream_error_detail: '{"error":"schema rejected"}'
    })

    const events = resolveDisplayUpstreamEvents(detail)

    expect(events).toHaveLength(1)
    expect(events[0]).toMatchObject({
      key: 'fallback',
      account_id: 101,
      account_name: 'openai-upstream-01',
      platform: 'openai',
      status_code: 422,
      request_id: 'gw-rid-1',
      kind: 'http_error',
      message: 'schema rejected'
    })
    expect(events[0].response_preview).toBe('{"error":"schema rejected"}')
  })

  it('reports upstream attribution when account or upstream context exists', () => {
    expect(hasUpstreamAttribution(makeDetail({ account_id: 101 }))).toBe(true)
    expect(hasUpstreamAttribution(makeDetail({ account_id: null, upstream_status_code: 400 }))).toBe(true)
    expect(hasUpstreamAttribution(makeDetail({ account_id: null, upstream_errors: '[{"upstream_status_code":400}]' }))).toBe(true)
    expect(hasUpstreamAttribution(makeDetail({
      account_id: null,
      upstream_status_code: null,
      upstream_error_message: '',
      upstream_error_detail: '',
      upstream_errors: ''
    }))).toBe(false)
  })
})
```

- [ ] **Step 2: Run utility tests and verify they fail**

Run:

```bash
cd /mnt/d/wanwan/project/self/APIStation/sub2api/frontend
pnpm test:run src/views/admin/ops/utils/__tests__/errorDetailUpstream.spec.ts
```

Expected: FAIL because `errorDetailUpstream.ts` does not exist.

- [ ] **Step 3: Implement upstream display parsing**

Create `frontend/src/views/admin/ops/utils/errorDetailUpstream.ts`:

```ts
import type { OpsErrorDetail } from '@/api/admin/ops'

export type DisplayUpstreamEvent = {
  key: string
  account_id?: number | null
  account_name?: string
  platform?: string
  status_code?: number | null
  request_id?: string
  kind?: string
  message?: string
  detail?: string
  response_preview?: string
}

type RawUpstreamEvent = {
  account_id?: number
  account_name?: string
  platform?: string
  upstream_status_code?: number
  upstream_request_id?: string
  kind?: string
  message?: string
  detail?: string
  upstream_response_body?: string
}

function clean(value: unknown): string {
  return typeof value === 'string' ? value.trim() : ''
}

function parseRawEvents(raw: string | undefined): RawUpstreamEvent[] {
  const text = clean(raw)
  if (!text || text === '[]' || text === '{}' || text.toLowerCase() === 'null') return []
  try {
    const parsed = JSON.parse(text)
    return Array.isArray(parsed) ? parsed.filter((item): item is RawUpstreamEvent => !!item && typeof item === 'object') : []
  } catch {
    return []
  }
}

export function resolveDisplayUpstreamEvents(detail: OpsErrorDetail | null | undefined): DisplayUpstreamEvent[] {
  if (!detail) return []

  const inline = parseRawEvents(detail.upstream_errors).map((ev, index) => ({
    key: `inline-${index}`,
    account_id: ev.account_id ?? null,
    account_name: clean(ev.account_name),
    platform: clean(ev.platform),
    status_code: ev.upstream_status_code ?? null,
    request_id: clean(ev.upstream_request_id),
    kind: clean(ev.kind),
    message: clean(ev.message),
    detail: clean(ev.detail),
    response_preview: clean(ev.upstream_response_body) || clean(ev.detail)
  }))

  if (inline.length > 0) return inline

  const status = detail.upstream_status_code ?? null
  const message = clean(detail.upstream_error_message)
  const detailPayload = clean(detail.upstream_error_detail)
  const hasFallback = detail.account_id != null || status != null || message || detailPayload
  if (!hasFallback) return []

  return [{
    key: 'fallback',
    account_id: detail.account_id ?? null,
    account_name: clean(detail.account_name),
    platform: clean(detail.platform),
    status_code: status,
    request_id: clean(detail.request_id) || clean(detail.client_request_id),
    kind: 'http_error',
    message,
    detail: detailPayload,
    response_preview: detailPayload || message
  }]
}

export function hasUpstreamAttribution(detail: OpsErrorDetail | null | undefined): boolean {
  return resolveDisplayUpstreamEvents(detail).length > 0
}
```

- [ ] **Step 4: Run utility tests and verify they pass**

Run:

```bash
cd /mnt/d/wanwan/project/self/APIStation/sub2api/frontend
pnpm test:run src/views/admin/ops/utils/__tests__/errorDetailUpstream.spec.ts
```

Expected: PASS.

- [ ] **Step 5: Commit frontend utility**

```bash
git add -- frontend/src/views/admin/ops/utils/errorDetailUpstream.ts frontend/src/views/admin/ops/utils/__tests__/errorDetailUpstream.spec.ts
git commit -m "test: add upstream error detail parsing"
```

### Task 5: Update Ops Error Detail Modal

**Files:**

- Modify: `frontend/src/views/admin/ops/components/OpsErrorDetailModal.vue`
- Create: `frontend/src/views/admin/ops/components/__tests__/OpsErrorDetailModal.spec.ts`

- [ ] **Step 1: Add failing component tests**

Create `frontend/src/views/admin/ops/components/__tests__/OpsErrorDetailModal.spec.ts`:

```ts
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import OpsErrorDetailModal from '../OpsErrorDetailModal.vue'

const mockGetRequestErrorDetail = vi.fn()
const mockListRequestErrorUpstreamErrors = vi.fn()

vi.mock('@/api/admin/ops', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/api/admin/ops')>()
  return {
    ...actual,
    opsAPI: {
      ...actual.opsAPI,
      getRequestErrorDetail: (...args: any[]) => mockGetRequestErrorDetail(...args),
      listRequestErrorUpstreamErrors: (...args: any[]) => mockListRequestErrorUpstreamErrors(...args)
    }
  }
})

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    showError: vi.fn()
  })
}))

describe('OpsErrorDetailModal', () => {
  beforeEach(() => {
    mockGetRequestErrorDetail.mockReset()
    mockListRequestErrorUpstreamErrors.mockReset()
    mockListRequestErrorUpstreamErrors.mockResolvedValue({ items: [], total: 0, page: 1, page_size: 100 })
  })

  it('shows user and upstream account for a request error with inline upstream events', async () => {
    mockGetRequestErrorDetail.mockResolvedValue({
      id: 7,
      created_at: '2026-05-12T10:00:00Z',
      phase: 'upstream',
      type: 'invalid_request_error',
      error_owner: 'provider',
      error_source: 'upstream_http',
      severity: 'P3',
      status_code: 400,
      platform: 'openai',
      model: 'gpt-4o-mini',
      is_retryable: false,
      retry_count: 0,
      resolved: false,
      client_request_id: 'client-rid-1',
      request_id: 'gw-rid-1',
      message: 'openai_error',
      user_id: 55,
      user_email: 'user@example.com',
      account_id: 101,
      account_name: 'openai-upstream-01',
      group_name: 'default',
      request_path: '/v1/chat/completions',
      stream: false,
      inbound_endpoint: '/v1/chat/completions',
      upstream_endpoint: '/v1/chat/completions',
      requested_model: 'gpt-4o-mini',
      upstream_model: 'gpt-4o-mini',
      error_body: '{"error":{"message":"openai_error","type":"invalid_request_error"}}',
      user_agent: 'test-client',
      upstream_status_code: 400,
      upstream_error_message: 'openai_error',
      upstream_error_detail: '{"error":{"message":"openai_error"}}',
      upstream_errors: JSON.stringify([
        {
          account_id: 101,
          account_name: 'openai-upstream-01',
          upstream_status_code: 400,
          upstream_request_id: 'req-upstream-1',
          kind: 'http_error',
          message: 'openai_error',
          detail: '{"error":{"message":"openai_error"}}'
        }
      ]),
      request_body: '',
      request_body_truncated: false,
      is_business_limited: false
    })

    const wrapper = mount(OpsErrorDetailModal, {
      props: { show: true, errorId: 7, errorType: 'request' },
      global: {
        stubs: {
          BaseDialog: { template: '<div><slot /></div>' },
          Icon: { template: '<span />' }
        }
      }
    })

    await flushPromises()

    expect(wrapper.text()).toContain('user@example.com')
    expect(wrapper.text()).toContain('openai-upstream-01')
    expect(wrapper.text()).toContain('101')
    expect(wrapper.text()).toContain('req-upstream-1')
    expect(wrapper.text()).toContain('openai_error')
  })
})
```

- [ ] **Step 2: Run component test and verify it fails**

Run:

```bash
cd /mnt/d/wanwan/project/self/APIStation/sub2api/frontend
pnpm test:run src/views/admin/ops/components/__tests__/OpsErrorDetailModal.spec.ts
```

Expected: FAIL because the modal still displays either user or account and does not render inline upstream events.

- [ ] **Step 3: Import upstream display helpers in the modal**

In `frontend/src/views/admin/ops/components/OpsErrorDetailModal.vue`, replace the current helper import block with:

```ts
import { resolvePrimaryResponseBody, resolveUpstreamPayload } from '../utils/errorDetailResponse'
import { hasUpstreamAttribution, resolveDisplayUpstreamEvents, type DisplayUpstreamEvent } from '../utils/errorDetailUpstream'
```

- [ ] **Step 4: Replace conditional user/account summary with separate cards**

Replace the current summary card that chooses between account and user with two cards:

```vue
<div class="rounded-xl bg-gray-50 p-4 dark:bg-dark-900">
  <div class="text-xs font-bold uppercase tracking-wider text-gray-400">{{ t('admin.ops.errorDetail.user') }}</div>
  <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">
    {{ detail.user_email || (detail.user_id != null ? String(detail.user_id) : '—') }}
  </div>
</div>

<div v-if="hasUpstreamContext" class="rounded-xl bg-gray-50 p-4 dark:bg-dark-900">
  <div class="text-xs font-bold uppercase tracking-wider text-gray-400">{{ t('admin.ops.errorDetail.account') }}</div>
  <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">
    {{ detail.account_name || (detail.account_id != null ? String(detail.account_id) : '—') }}
  </div>
</div>
```

- [ ] **Step 5: Render inline upstream events before correlated upstream logs**

In the script section, replace the old `correlatedUpstreamErrors` computed and expanded set types with:

```ts
const inlineUpstreamEvents = computed<DisplayUpstreamEvent[]>(() => resolveDisplayUpstreamEvents(detail.value))
const hasUpstreamContext = computed(() => hasUpstreamAttribution(detail.value))

function correlatedDetailToEvent(ev: OpsErrorDetail): DisplayUpstreamEvent {
  return {
    key: `correlated-${ev.id}`,
    account_id: ev.account_id ?? null,
    account_name: ev.account_name,
    platform: ev.platform,
    status_code: ev.upstream_status_code ?? ev.status_code ?? null,
    request_id: ev.request_id || ev.client_request_id,
    kind: ev.type,
    message: ev.message,
    detail: ev.upstream_error_detail || ev.error_body,
    response_preview: getUpstreamResponsePreview(ev)
  }
}

const displayedUpstreamEvents = computed<DisplayUpstreamEvent[]>(() => {
  if (inlineUpstreamEvents.value.length > 0) return inlineUpstreamEvents.value
  return correlatedUpstream.value.map(correlatedDetailToEvent)
})

const expandedUpstreamDetailIds = ref(new Set<string>())
```

Then update the template loop:

```vue
<div v-if="!correlatedUpstreamLoading && !displayedUpstreamEvents.length" class="mt-3 text-sm text-gray-500 dark:text-gray-400">
  {{ t('common.noData') }}
</div>

<div v-else class="mt-4 space-y-3">
  <div
    v-for="(ev, idx) in displayedUpstreamEvents"
    :key="ev.key"
    class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800"
  >
```

Update event field usage inside the loop:

```vue
<span v-if="ev.kind" class="ml-2 rounded-md bg-gray-100 px-2 py-0.5 font-mono text-[10px] font-bold text-gray-700 dark:bg-dark-700 dark:text-gray-200">{{ ev.kind }}</span>
```

```vue
{{ ev.status_code ?? '—' }}
```

```vue
<span class="ml-1 font-mono">{{ ev.request_id || '—' }}</span>
```

```vue
<div v-if="ev.account_name || ev.account_id != null">
  <span class="text-gray-400">{{ t('admin.ops.errorDetail.account') }}:</span>
  <span class="ml-1 font-mono">{{ ev.account_name || ev.account_id }}</span>
</div>
```

```vue
<button
  type="button"
  class="inline-flex items-center gap-1.5 rounded-md px-1.5 py-1 text-[10px] font-bold text-primary-700 hover:bg-primary-50 disabled:cursor-not-allowed disabled:opacity-60 dark:text-primary-200 dark:hover:bg-dark-700"
  :disabled="!ev.response_preview"
  :title="ev.response_preview ? '' : t('common.noData')"
  @click="toggleUpstreamDetail(ev.key)"
>
```

```vue
<pre
  v-if="expandedUpstreamDetailIds.has(ev.key)"
  class="mt-3 max-h-[240px] overflow-auto rounded-xl border border-gray-200 bg-gray-50 p-3 text-xs text-gray-800 dark:border-dark-700 dark:bg-dark-900 dark:text-gray-100"
><code>{{ prettyJSON(ev.response_preview) }}</code></pre>
```

Update `toggleUpstreamDetail`:

```ts
function toggleUpstreamDetail(id: string) {
  const next = new Set(expandedUpstreamDetailIds.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  expandedUpstreamDetailIds.value = next
}
```

Update the watcher reset:

```ts
expandedUpstreamDetailIds.value = new Set<string>()
```

- [ ] **Step 6: Run frontend tests**

Run:

```bash
cd /mnt/d/wanwan/project/self/APIStation/sub2api/frontend
pnpm test:run src/views/admin/ops/utils/__tests__/errorDetailUpstream.spec.ts src/views/admin/ops/components/__tests__/OpsErrorDetailModal.spec.ts src/views/admin/ops/utils/__tests__/errorDetailResponse.spec.ts
```

Expected: PASS.

- [ ] **Step 7: Commit frontend detail implementation**

```bash
git add -- frontend/src/views/admin/ops/components/OpsErrorDetailModal.vue frontend/src/views/admin/ops/components/__tests__/OpsErrorDetailModal.spec.ts
git commit -m "fix: show upstream account in error details"
```

### Task 6: Final Verification

**Files:**

- No file changes expected.

- [ ] **Step 1: Run targeted backend verification**

Run:

```bash
cd /mnt/d/wanwan/project/self/APIStation/sub2api/backend
go test ./internal/handler ./internal/repository -run 'TestOps|Test.*AccountRequestStats|Test.*SuccessRate' -count=1
```

Expected: PASS.

- [ ] **Step 2: Run targeted frontend verification**

Run:

```bash
cd /mnt/d/wanwan/project/self/APIStation/sub2api/frontend
pnpm test:run src/views/admin/ops/utils/__tests__/errorDetailUpstream.spec.ts src/views/admin/ops/utils/__tests__/errorDetailResponse.spec.ts src/views/admin/ops/components/__tests__/OpsErrorDetailModal.spec.ts
```

Expected: PASS.

- [ ] **Step 3: Run static checks if dependencies are already installed**

Run:

```bash
cd /mnt/d/wanwan/project/self/APIStation/sub2api/frontend
pnpm typecheck
```

Expected: PASS. If this fails because frontend dependencies are absent, stop and report that dependency installation is needed before typecheck can run.

- [ ] **Step 4: Review final git state**

Run:

```bash
cd /mnt/d/wanwan/project/self/APIStation/sub2api
git status --short
```

Expected: no unstaged changes from this plan. Existing unrelated untracked files such as `.qoder/` may remain.
