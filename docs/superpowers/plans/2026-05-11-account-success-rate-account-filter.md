# Dashboard Account Success Rate Filter Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a single-select account filter to the admin dashboard "账号请求成功率趋势" card, with a default "全部账号" option and candidates limited to currently normal accounts.

**Architecture:** Extend the existing `account-success-rate-trend` API with an optional `account_id` filter while preserving the current aggregate behavior when omitted. On the frontend, load the normal-account options from the existing admin accounts list API, keep the chart component presentation-focused, and let `DashboardView` own the selected account state and data fetching.

**Tech Stack:** Go, Gin, repository-backed dashboard service, Vue 3, Vitest, existing admin API modules and chart component.

---

### Task 1: Lock backend behavior with failing tests

**Files:**
- Modify: `backend/internal/handler/admin/dashboard_success_rate_handler_test.go`

- [ ] **Step 1: Write the failing handler/cache test for `account_id` passthrough**

```go
func TestDashboardHandler_GetAccountSuccessRateTrend_AccountIDAffectsCacheKeyAndRepoCall(t *testing.T) {
    dashboardSuccessRateTrendCache = newSnapshotCache(30 * time.Second)
    t.Cleanup(func() {
        dashboardSuccessRateTrendCache = newSnapshotCache(30 * time.Second)
    })

    gin.SetMode(gin.TestMode)
    repo := &accountSuccessRateRepoProbe{}
    dashboardSvc := service.NewDashboardService(repo, nil, nil, nil)
    handler := NewDashboardHandler(dashboardSvc, nil)
    router := gin.New()
    router.GET("/admin/dashboard/account-success-rate-trend", handler.GetAccountSuccessRateTrend)

    req1 := httptest.NewRequest(http.MethodGet, "/admin/dashboard/account-success-rate-trend?start_date=2026-03-01&end_date=2026-03-01&granularity=1h&account_id=42", nil)
    rec1 := httptest.NewRecorder()
    router.ServeHTTP(rec1, req1)

    req2 := httptest.NewRequest(http.MethodGet, "/admin/dashboard/account-success-rate-trend?start_date=2026-03-01&end_date=2026-03-01&granularity=1h&account_id=7", nil)
    rec2 := httptest.NewRecorder()
    router.ServeHTTP(rec2, req2)

    require.Equal(t, int32(2), repo.trendCalls.Load())
    require.Equal(t, int64(7), repo.lastTrendAccountID)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /mnt/d/wanwan/project/self/APIStation/sub2api/backend && go test ./internal/handler/admin -run 'TestDashboardHandler_GetAccountSuccessRateTrend_'`

Expected: FAIL because the probe/service/handler do not yet support `account_id`.

- [ ] **Step 3: Write the failing repository-facing expectation**

```go
func (r *accountSuccessRateRepoProbe) GetAccountSuccessRateTrend(
    ctx context.Context,
    startTime, endTime time.Time,
    granularity string,
    userTZ string,
    accountID int64,
) (*usagestats.AccountSuccessRateTrendResponse, error) {
    r.lastTrendAccountID = accountID
    ...
}
```

- [ ] **Step 4: Re-run the same test**

Run: `cd /mnt/d/wanwan/project/self/APIStation/sub2api/backend && go test ./internal/handler/admin -run 'TestDashboardHandler_GetAccountSuccessRateTrend_'`

Expected: FAIL in production code compile/use sites, proving the contract is now red.

### Task 2: Implement backend account filter support

**Files:**
- Modify: `backend/internal/service/dashboard_service.go`
- Modify: `backend/internal/handler/admin/dashboard_handler.go`
- Modify: `backend/internal/repository/usage_log_repo.go`

- [ ] **Step 1: Add the optional `accountID` parameter through service and handler**

```go
type accountSuccessRateTrendReader interface {
    GetAccountSuccessRateTrend(ctx context.Context, startTime, endTime time.Time, granularity string, userTZ string, accountID int64) (*usagestats.AccountSuccessRateTrendResponse, error)
}
```

```go
if accountIDStr := c.Query("account_id"); accountIDStr != "" {
    if id, err := strconv.ParseInt(accountIDStr, 10, 64); err == nil {
        accountID = id
    }
}
```

- [ ] **Step 2: Include `account_id` in the dashboard success-rate cache key**

```go
type accountSuccessRateTrendCacheKey struct {
    StartTime   string `json:"start_time"`
    EndTime     string `json:"end_time"`
    Granularity string `json:"granularity"`
    Timezone    string `json:"timezone"`
    AccountID   int64  `json:"account_id,omitempty"`
}
```

- [ ] **Step 3: Apply the repository SQL filter only when a positive `account_id` is provided**

```go
if accountID > 0 {
    query += " AND ars.account_id = $5"
    rows, err = r.sql.QueryContext(ctx, query, startUTC, endUTC, tzName, service.StatusActive, accountID)
} else {
    rows, err = r.sql.QueryContext(ctx, query, startUTC, endUTC, tzName, service.StatusActive)
}
```

- [ ] **Step 4: Run backend tests**

Run: `cd /mnt/d/wanwan/project/self/APIStation/sub2api/backend && go test ./internal/handler/admin -run 'TestDashboardHandler_GetAccountSuccessRateTrend_'`

Expected: PASS

### Task 3: Lock frontend filter behavior with failing tests

**Files:**
- Modify: `frontend/src/views/admin/__tests__/DashboardView.spec.ts`
- Modify: `frontend/src/components/charts/__tests__/AccountSuccessRateTrend.spec.ts`

- [ ] **Step 1: Write the failing dashboard view test for loading normal-account options**

```ts
expect(list).toHaveBeenCalledWith(1, 200, { status: 'active', lite: 'true' }, undefined)
```

```ts
expect(getAccountSuccessRateTrend).toHaveBeenCalledWith(expect.objectContaining({
  account_id: 42
}))
```

- [ ] **Step 2: Write the failing chart component test for the selector UI**

```ts
expect(wrapper.text()).toContain('All Accounts')
expect(wrapper.text()).toContain('claude-01')
```

- [ ] **Step 3: Run frontend tests to verify they fail**

Run: `cd /mnt/d/wanwan/project/self/APIStation/sub2api/frontend && pnpm test:run src/views/admin/__tests__/DashboardView.spec.ts src/components/charts/__tests__/AccountSuccessRateTrend.spec.ts`

Expected: FAIL because the dashboard view and chart component do not yet expose account filtering.

### Task 4: Implement frontend selector, API wiring, and copy

**Files:**
- Modify: `frontend/src/api/admin/dashboard.ts`
- Modify: `frontend/src/views/admin/DashboardView.vue`
- Modify: `frontend/src/components/charts/AccountSuccessRateTrend.vue`
- Modify: `frontend/src/i18n/locales/zh.ts`
- Modify: `frontend/src/i18n/locales/en.ts`

- [ ] **Step 1: Extend the trend API params with optional `account_id`**

```ts
export interface AccountSuccessRateTrendParams {
  start_date?: string
  end_date?: string
  granularity?: '10m' | '1h' | '1d'
  timezone?: string
  account_id?: number
}
```

- [ ] **Step 2: In `DashboardView`, load normal-account options from admin accounts**

```ts
const accountOptions = ref([{ value: 0, label: t('admin.dashboard.allAccounts') }])
const selectedAccountId = ref(0)
```

```ts
const response = await adminAPI.accounts.list(1, 200, { status: 'active', lite: 'true' })
accountOptions.value = [
  { value: 0, label: t('admin.dashboard.allAccounts') },
  ...response.items.filter((account) => account.schedulable).map((account) => ({
    value: account.id,
    label: account.name
  }))
]
```

- [ ] **Step 3: Pass selector props/events through the chart component and refresh on change**

```ts
const onSuccessRateAccountChange = (value: number) => {
  selectedAccountId.value = value
  loadSuccessRateTrend()
}
```

```vue
<AccountSuccessRateTrend
  :account-options="accountOptions"
  :selected-account-id="selectedAccountId"
  @update:selected-account-id="onSuccessRateAccountChange"
/>
```

- [ ] **Step 4: Add i18n copy for the default option and label**

```ts
allAccounts: '全部账号',
accountFilter: '账号'
```

- [ ] **Step 5: Run the targeted frontend tests**

Run: `cd /mnt/d/wanwan/project/self/APIStation/sub2api/frontend && pnpm test:run src/views/admin/__tests__/DashboardView.spec.ts src/components/charts/__tests__/AccountSuccessRateTrend.spec.ts`

Expected: PASS

### Task 5: Final verification

**Files:**
- Modify: `docs/superpowers/specs/2026-05-11-account-success-rate-design.md`
- Create: `docs/superpowers/plans/2026-05-11-account-success-rate-account-filter.md`

- [ ] **Step 1: Re-run backend verification**

Run: `cd /mnt/d/wanwan/project/self/APIStation/sub2api/backend && go test ./internal/handler/admin -run 'TestDashboardHandler_GetAccountSuccessRateTrend_'`

Expected: PASS

- [ ] **Step 2: Re-run frontend verification**

Run: `cd /mnt/d/wanwan/project/self/APIStation/sub2api/frontend && pnpm test:run src/views/admin/__tests__/DashboardView.spec.ts src/components/charts/__tests__/AccountSuccessRateTrend.spec.ts`

Expected: PASS

- [ ] **Step 3: Sanity-check changed files**

Run: `cd /mnt/d/wanwan/project/self/APIStation/sub2api && git diff -- backend/internal/handler/admin/dashboard_success_rate_handler_test.go backend/internal/handler/admin/dashboard_handler.go backend/internal/repository/usage_log_repo.go backend/internal/service/dashboard_service.go frontend/src/api/admin/dashboard.ts frontend/src/views/admin/DashboardView.vue frontend/src/components/charts/AccountSuccessRateTrend.vue frontend/src/views/admin/__tests__/DashboardView.spec.ts frontend/src/components/charts/__tests__/AccountSuccessRateTrend.spec.ts frontend/src/i18n/locales/zh.ts frontend/src/i18n/locales/en.ts`

Expected: Only the planned account-filter changes appear, with no unrelated reverts.
