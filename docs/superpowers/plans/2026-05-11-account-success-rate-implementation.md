# Account Success Rate Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add aggregate-backed raw request success-rate statistics to the admin dashboard and admin accounts list without scanning `usage_logs` or `ops_error_logs` on normal page loads.

**Architecture:** Reuse the existing dashboard aggregation scheduler to maintain a new `account_request_stats_10m` table from `usage_logs` and `ops_error_logs`. Keep read APIs inside the existing admin dashboard/account flows by extending `UsageLogRepository` with aggregate-table readers, then expose one new dashboard trend endpoint and one new accounts batch endpoint for the frontend.

**Tech Stack:** Go, Gin, PostgreSQL, existing dashboard aggregation service, Vue 3, Vite, Vitest, `vue-chartjs`

---

## File Structure

**Create**

- `backend/migrations/136_add_account_request_stats_10m.sql`: creates the new 10-minute aggregate table plus indexes.
- `backend/internal/repository/account_request_stats_integration_test.go`: integration coverage for the new aggregate write/read paths.
- `backend/internal/handler/admin/account_success_rate_cache.go`: cache key builder + snapshot cache for the batch account success-rate endpoint.
- `backend/internal/handler/admin/dashboard_success_rate_handler_test.go`: unit coverage for the new dashboard trend endpoint and batch account handler.
- `frontend/src/components/charts/AccountSuccessRateTrend.vue`: dashboard chart for `10m | 1h | 1d` success-rate points with per-account tooltip details.
- `frontend/src/components/account/AccountSuccessRateCell.vue`: compact account-list cell showing rate plus success/failure counts.
- `frontend/src/components/charts/__tests__/AccountSuccessRateTrend.spec.ts`: chart empty/loading/data-state tests.
- `frontend/src/components/account/__tests__/AccountSuccessRateCell.spec.ts`: cell rendering tests for populated, empty, and degraded states.
- `frontend/src/views/admin/__tests__/AccountsView.successRate.spec.ts`: page-level non-blocking batch-loading tests for the new account-list column.

**Modify**

- `backend/internal/pkg/usagestats/usage_log_types.go`: shared response structs and new dashboard/account success-rate fields.
- `backend/internal/service/account_usage_service.go`: batch account success-rate read method for the accounts page.
- `backend/internal/service/dashboard_service.go`: dashboard success-rate snapshot + trend read methods.
- `backend/internal/repository/dashboard_aggregation_repo.go`: compute, recompute, and clean up `account_request_stats_10m`.
- `backend/internal/repository/usage_log_repo.go`: read aggregate-table snapshot/trend/batch data without raw-log scans.
- `backend/internal/repository/migrations_schema_integration_test.go`: schema/index assertions for the new table.
- `backend/internal/handler/admin/dashboard_snapshot_v2_handler.go`: include new success-rate fields in `snapshot-v2`.
- `backend/internal/handler/admin/dashboard_handler.go`: add `GET /api/v1/admin/dashboard/account-success-rate-trend`.
- `backend/internal/handler/admin/account_handler.go`: add `POST /api/v1/admin/accounts/success-rate/batch`.
- `backend/internal/handler/admin/id_list_utils_test.go`: cache-key coverage for the new batch cache helper.
- `backend/internal/server/routes/admin.go`: register the two new endpoints.
- `backend/internal/server/api_contract_test.go`: update the usage-log stub and add contract cases for the new responses.
- `frontend/src/types/index.ts`: success-rate interfaces and dashboard/account type extensions.
- `frontend/src/api/admin/dashboard.ts`: trend endpoint types + client method.
- `frontend/src/api/admin/accounts.ts`: batch success-rate request/response types + client method.
- `frontend/src/views/admin/DashboardView.vue`: replace the first card and add the success-rate trend card.
- `frontend/src/views/admin/AccountsView.vue`: add the success-rate column and non-blocking batch fetch flow.
- `frontend/src/views/admin/__tests__/DashboardView.spec.ts`: success-rate snapshot/trend request coverage.
- `frontend/src/views/admin/__tests__/AccountsView.bulkEdit.spec.ts`: update existing stubs if shared imports/types change.
- `frontend/src/components/account/index.ts`: export the new success-rate cell.
- `frontend/src/i18n/locales/en.ts`: English labels and empty/degraded copy.
- `frontend/src/i18n/locales/zh.ts`: Chinese labels and empty/degraded copy.

## Data Model Notes

- `account_request_stats_10m.bucket_start` is stored in UTC and always aligned to a 10-minute boundary.
- `success_count` comes from `usage_logs` grouped by `(bucket_start, account_id)`.
- `failed_count` comes from `ops_error_logs` where `status_code >= 400`, `account_id IS NOT NULL`, and `is_count_tokens = FALSE`.
- Successful `count_tokens` probes should also be excluded by filtering `usage_logs.inbound_endpoint` values ending in `/count_tokens`.
- `request_count` is stored as `success_count + failed_count`.
- `success_rate` is never stored; compute it as `success_count * 100.0 / request_count`, and return `null` when `request_count = 0`.
- The dashboard snapshot counts all accounts in the retention window.
- The dashboard trend filters to accounts that are currently normal at query time: `status = 'active' AND schedulable = TRUE`.

### Task 1: Create The Aggregate Table And Shared Contracts

**Files:**
- Create: `backend/migrations/136_add_account_request_stats_10m.sql`
- Modify: `backend/internal/repository/migrations_schema_integration_test.go`
- Modify: `backend/internal/pkg/usagestats/usage_log_types.go`
- Modify: `backend/internal/service/account_usage_service.go`
- Modify: `backend/internal/service/dashboard_service.go`
- Test: `backend/internal/repository/migrations_schema_integration_test.go`
- Test: `backend/internal/server/api_contract_test.go`

- [x] **Step 1: Write the failing schema assertions**

```go
// backend/internal/repository/migrations_schema_integration_test.go
var accountRequestStatsRegclass sql.NullString
require.NoError(t, tx.QueryRowContext(context.Background(), "SELECT to_regclass('public.account_request_stats_10m')").Scan(&accountRequestStatsRegclass))
require.True(t, accountRequestStatsRegclass.Valid, "expected account_request_stats_10m table to exist")

requireColumn(t, tx, "account_request_stats_10m", "bucket_start", "timestamp with time zone", 0, false)
requireColumn(t, tx, "account_request_stats_10m", "account_id", "bigint", 0, false)
requireColumn(t, tx, "account_request_stats_10m", "success_count", "bigint", 0, false)
requireColumn(t, tx, "account_request_stats_10m", "failed_count", "bigint", 0, false)
requireColumn(t, tx, "account_request_stats_10m", "request_count", "bigint", 0, false)
requireColumn(t, tx, "account_request_stats_10m", "computed_at", "timestamp with time zone", 0, false)
requireIndex(t, tx, "account_request_stats_10m", "account_request_stats_10m_bucket_account_key")
requireIndex(t, tx, "account_request_stats_10m", "idx_account_request_stats_10m_account_bucket_desc")
requireIndex(t, tx, "account_request_stats_10m", "idx_account_request_stats_10m_bucket_desc")
```

- [x] **Step 2: Run the schema test and verify it fails**

Run:

```bash
cd backend && go test -tags=integration ./internal/repository -run TestMigrationsRunner_IsIdempotent_AndSchemaIsUpToDate -count=1
```

Expected: FAIL with a missing-table assertion for `account_request_stats_10m`.

- [x] **Step 3: Add the migration**

```sql
-- backend/migrations/136_add_account_request_stats_10m.sql
CREATE TABLE IF NOT EXISTS account_request_stats_10m (
    bucket_start   TIMESTAMPTZ NOT NULL,
    account_id     BIGINT      NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    success_count  BIGINT      NOT NULL DEFAULT 0,
    failed_count   BIGINT      NOT NULL DEFAULT 0,
    request_count  BIGINT      NOT NULL DEFAULT 0,
    computed_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (bucket_start, account_id),
    CONSTRAINT account_request_stats_10m_request_count_check
        CHECK (request_count = success_count + failed_count)
);

CREATE INDEX IF NOT EXISTS idx_account_request_stats_10m_account_bucket_desc
    ON account_request_stats_10m (account_id, bucket_start DESC);

CREATE INDEX IF NOT EXISTS idx_account_request_stats_10m_bucket_desc
    ON account_request_stats_10m (bucket_start DESC);
```

- [x] **Step 4: Add shared Go contracts before any handler/repository work**

```go
// backend/internal/pkg/usagestats/usage_log_types.go
type SuccessRateSummary struct {
    SuccessCount int64    `json:"success_count"`
    FailedCount  int64    `json:"failed_count"`
    RequestCount int64    `json:"request_count"`
    SuccessRate  *float64 `json:"success_rate"`
}

type AccountSuccessRateTrendAccount struct {
    AccountID     int64   `json:"account_id"`
    AccountName   string  `json:"account_name"`
    SuccessCount  int64   `json:"success_count"`
    FailedCount   int64   `json:"failed_count"`
    RequestCount  int64   `json:"request_count"`
    SuccessRate   float64 `json:"success_rate"`
}

type AccountSuccessRateTrendPoint struct {
    BucketStart   string                           `json:"bucket_start"`
    SuccessCount  int64                            `json:"success_count"`
    FailedCount   int64                            `json:"failed_count"`
    RequestCount  int64                            `json:"request_count"`
    SuccessRate   float64                          `json:"success_rate"`
    Accounts      []AccountSuccessRateTrendAccount `json:"accounts"`
}

type AccountSuccessRateTrendResponse struct {
    Bucket     string                        `json:"bucket"`
    ComputedAt string                        `json:"computed_at"`
    Stale      bool                          `json:"stale"`
    Partial    bool                          `json:"partial"`
    Points     []AccountSuccessRateTrendPoint `json:"points"`
}
```

```go
// also extend DashboardStats in the same file
TodaySuccessCount  int64    `json:"today_success_count"`
TodayFailedCount   int64    `json:"today_failed_count"`
TodaySuccessRate   *float64 `json:"today_success_rate"`
HistorySuccessRate *float64 `json:"history_success_rate"`
```

```go
// backend/internal/service/account_usage_service.go
func (s *AccountUsageService) GetHistorySuccessRateBatch(ctx context.Context, accountIDs []int64) (map[int64]*usagestats.SuccessRateSummary, error)
```

```go
// backend/internal/service/dashboard_service.go
func (s *DashboardService) GetAccountSuccessRateTrend(
    ctx context.Context,
    startTime, endTime time.Time,
    granularity string,
    userTZ string,
) (*usagestats.AccountSuccessRateTrendResponse, error)
```

- [x] **Step 5: Re-run schema tests and compile contracts, then commit**

Run:

```bash
cd backend && go test -tags=integration ./internal/repository -run TestMigrationsRunner_IsIdempotent_AndSchemaIsUpToDate -count=1
cd backend && go test -tags=unit ./internal/service ./internal/server -run 'TestAPIContracts|TestDashboardService' -count=1
```

Expected: PASS for the schema test; compile should still fail later because the new repository methods are not implemented yet.

- [x] **Step 6: Commit**

```bash
git add backend/migrations/136_add_account_request_stats_10m.sql \
        backend/internal/repository/migrations_schema_integration_test.go \
        backend/internal/pkg/usagestats/usage_log_types.go \
        backend/internal/service/account_usage_service.go \
        backend/internal/service/dashboard_service.go
git commit -m "feat: add account request stats schema and contracts"
```

### Task 2: Aggregate And Retain 10-Minute Account Success Stats

**Files:**
- Modify: `backend/internal/repository/dashboard_aggregation_repo.go`
- Create: `backend/internal/repository/account_request_stats_integration_test.go`
- Modify: `backend/internal/service/dashboard_aggregation_service.go`
- Test: `backend/internal/repository/account_request_stats_integration_test.go`
- Test: `backend/internal/service/dashboard_aggregation_service_test.go`

- [x] **Step 1: Write the failing aggregation tests**

```go
// backend/internal/repository/account_request_stats_integration_test.go
func TestAccountRequestStats10m_AggregateRange_WritesSuccessAndFailureBuckets(t *testing.T) {
    tx := testTx(t)
    repo := newDashboardAggregationRepositoryWithSQL(tx)
    usageRepo := newUsageLogRepositoryWithSQL(&http.Client{Timeout: time.Second}, tx)

    start := time.Date(2026, 5, 11, 9, 0, 0, 0, time.UTC)
    end := start.Add(30 * time.Minute)

    _, err := tx.ExecContext(context.Background(), `
        INSERT INTO usage_logs (
            user_id, api_key_id, account_id, model, input_tokens, output_tokens,
            cache_creation_tokens, cache_read_tokens, total_cost, actual_cost,
            created_at, inbound_endpoint
        ) VALUES
            (1, 1, 101, 'claude-sonnet-4', 10, 20, 0, 0, 1, 1, $1, '/v1/messages'),
            (1, 1, 101, 'claude-sonnet-4', 10, 20, 0, 0, 1, 1, $2, '/v1/messages'),
            (1, 1, 101, 'claude-sonnet-4', 10, 20, 0, 0, 1, 1, $3, '/v1/messages/count_tokens')
    `, start.Add(2*time.Minute), start.Add(7*time.Minute), start.Add(8*time.Minute))
    require.NoError(t, err)

    _, err = tx.ExecContext(context.Background(), `
        INSERT INTO ops_error_logs (
            request_id, account_id, status_code, error_message, is_count_tokens, created_at
        ) VALUES
            ('req-fail-1', 101, 500, 'upstream failed', FALSE, $1),
            ('req-fail-2', NULL, 500, 'missing account', FALSE, $1),
            ('req-fail-3', 101, 429, 'probe failure', TRUE, $1)
    `, start.Add(5*time.Minute))
    require.NoError(t, err)

    require.NoError(t, repo.AggregateRange(context.Background(), start, end))

    rows, err := usageRepo.GetAccountSuccessRateBatch(context.Background(), []int64{101})
    require.NoError(t, err)
    require.Equal(t, int64(2), rows[101].SuccessCount)
    require.Equal(t, int64(1), rows[101].FailedCount)
    require.Equal(t, int64(3), rows[101].RequestCount)
}
```

```go
func TestAccountRequestStats10m_CleanupAggregates_DeletesExpiredBuckets(t *testing.T) {
    tx := testTx(t)
    repo := newDashboardAggregationRepositoryWithSQL(tx)

    _, err := tx.ExecContext(context.Background(), `
        INSERT INTO account_request_stats_10m (bucket_start, account_id, success_count, failed_count, request_count, computed_at)
        VALUES
            ('2026-02-01T00:00:00Z', 101, 4, 1, 5, NOW()),
            ('2026-05-01T00:00:00Z', 101, 3, 0, 3, NOW())
    `)
    require.NoError(t, err)

    require.NoError(t, repo.CleanupAggregates(context.Background(), time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)))

    var remaining int
    require.NoError(t, tx.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM account_request_stats_10m").Scan(&remaining))
    require.Equal(t, 1, remaining)
}
```

- [x] **Step 2: Run the new repository tests and verify they fail**

Run:

```bash
cd backend && go test -tags=integration ./internal/repository -run 'TestAccountRequestStats10m_' -count=1
```

Expected: FAIL because `AggregateRange` does not populate `account_request_stats_10m` yet.

- [x] **Step 3: Implement the writer in the aggregation repository**

```go
// backend/internal/repository/dashboard_aggregation_repo.go
func (r *dashboardAggregationRepository) aggregateRangeInTx(ctx context.Context, hourStart, hourEnd, dayStart, dayEnd time.Time) error {
    if err := r.upsertAccountRequestStats10m(ctx, hourStart, hourEnd); err != nil {
        return err
    }
    if err := r.insertHourlyActiveUsers(ctx, hourStart, hourEnd); err != nil {
        return err
    }
    if err := r.insertDailyActiveUsers(ctx, hourStart, hourEnd); err != nil {
        return err
    }
    if err := r.upsertHourlyAggregates(ctx, hourStart, hourEnd); err != nil {
        return err
    }
    return r.upsertDailyAggregates(ctx, dayStart, dayEnd)
}
```

```go
func (r *dashboardAggregationRepository) upsertAccountRequestStats10m(ctx context.Context, start, end time.Time) error {
    const q = `
WITH success_rows AS (
    SELECT
        date_bin('10 minutes', created_at, TIMESTAMPTZ '1970-01-01 00:00:00Z') AS bucket_start,
        account_id,
        COUNT(*) AS success_count
    FROM usage_logs
    WHERE created_at >= $1
      AND created_at < $2
      AND account_id IS NOT NULL
      AND COALESCE(inbound_endpoint, '') NOT LIKE '%/count_tokens'
    GROUP BY 1, 2
),
failure_rows AS (
    SELECT
        date_bin('10 minutes', created_at, TIMESTAMPTZ '1970-01-01 00:00:00Z') AS bucket_start,
        account_id,
        COUNT(*) AS failed_count
    FROM ops_error_logs
    WHERE created_at >= $1
      AND created_at < $2
      AND status_code >= 400
      AND account_id IS NOT NULL
      AND is_count_tokens = FALSE
    GROUP BY 1, 2
),
merged AS (
    SELECT
        COALESCE(s.bucket_start, f.bucket_start) AS bucket_start,
        COALESCE(s.account_id, f.account_id) AS account_id,
        COALESCE(s.success_count, 0) AS success_count,
        COALESCE(f.failed_count, 0) AS failed_count
    FROM success_rows s
    FULL OUTER JOIN failure_rows f
      ON s.bucket_start = f.bucket_start
     AND s.account_id = f.account_id
)
INSERT INTO account_request_stats_10m (
    bucket_start,
    account_id,
    success_count,
    failed_count,
    request_count,
    computed_at
)
SELECT
    bucket_start,
    account_id,
    success_count,
    failed_count,
    success_count + failed_count,
    NOW()
FROM merged
ON CONFLICT (bucket_start, account_id)
DO UPDATE SET
    success_count = EXCLUDED.success_count,
    failed_count = EXCLUDED.failed_count,
    request_count = EXCLUDED.request_count,
    computed_at = EXCLUDED.computed_at
`
    _, err := r.sql.ExecContext(ctx, q, start.UTC(), end.UTC())
    return err
}
```

```go
// cleanup path in the same file
if _, err := r.sql.ExecContext(ctx, "DELETE FROM account_request_stats_10m WHERE bucket_start < $1", hourlyCutoffUTC); err != nil {
    return err
}
```

- [x] **Step 4: Make recompute and retention cover the new table**

```go
// backend/internal/repository/dashboard_aggregation_repo.go
if _, err := r.sql.ExecContext(ctx, "DELETE FROM account_request_stats_10m WHERE bucket_start >= $1 AND bucket_start < $2", hourStart, hourEnd); err != nil {
    return err
}
```

```go
// keep DashboardAggregationService retention behavior unchanged;
// the repository CleanupAggregates implementation now clears the 10m table too.
```

- [x] **Step 5: Re-run aggregation tests and commit**

Run:

```bash
cd backend && go test -tags=integration ./internal/repository -run 'TestAccountRequestStats10m_|TestUsageLogRepoSuite/TestDashboardAggregationConsistency' -count=1
cd backend && go test -tags=unit ./internal/service -run 'TestDashboardAggregationService_' -count=1
```

Expected: PASS.

- [x] **Step 6: Commit**

```bash
git add backend/internal/repository/dashboard_aggregation_repo.go \
        backend/internal/repository/account_request_stats_integration_test.go \
        backend/internal/service/dashboard_aggregation_service.go
git commit -m "feat: aggregate account request success stats"
```

### Task 3: Expose Aggregate-Backed Backend Read APIs

**Files:**
- Modify: `backend/internal/repository/usage_log_repo.go`
- Modify: `backend/internal/service/dashboard_service.go`
- Modify: `backend/internal/service/account_usage_service.go`
- Modify: `backend/internal/handler/admin/dashboard_snapshot_v2_handler.go`
- Modify: `backend/internal/handler/admin/dashboard_handler.go`
- Modify: `backend/internal/handler/admin/account_handler.go`
- Create: `backend/internal/handler/admin/account_success_rate_cache.go`
- Create: `backend/internal/handler/admin/dashboard_success_rate_handler_test.go`
- Modify: `backend/internal/handler/admin/id_list_utils_test.go`
- Modify: `backend/internal/server/routes/admin.go`
- Modify: `backend/internal/server/api_contract_test.go`
- Test: `backend/internal/handler/admin/dashboard_success_rate_handler_test.go`
- Test: `backend/internal/server/api_contract_test.go`

- [x] **Step 1: Write failing handler tests for the new trend and batch endpoints**

```go
// backend/internal/handler/admin/dashboard_success_rate_handler_test.go
type dashboardUsageRepoCacheProbe struct {
    service.UsageLogRepository
    trendCalls          atomic.Int32
    successTrend        *usagestats.AccountSuccessRateTrendResponse
}

func (r *dashboardUsageRepoCacheProbe) GetAccountSuccessRateTrend(
    ctx context.Context,
    startTime, endTime time.Time,
    granularity string,
    userTZ string,
) (*usagestats.AccountSuccessRateTrendResponse, error) {
    r.trendCalls.Add(1)
    return r.successTrend, nil
}

func TestDashboardHandler_GetAccountSuccessRateTrend_UsesCache(t *testing.T) {
    gin.SetMode(gin.TestMode)
    repo := &dashboardUsageRepoCacheProbe{}
    repo.successTrend = &usagestats.AccountSuccessRateTrendResponse{
        Bucket: "1h",
        ComputedAt: "2026-05-11T10:00:00Z",
        Points: []usagestats.AccountSuccessRateTrendPoint{{
            BucketStart: "2026-05-11T09:00:00Z",
            SuccessCount: 12,
            FailedCount: 1,
            RequestCount: 13,
            SuccessRate: 92.31,
        }},
    }

    dashboardSvc := service.NewDashboardService(repo, nil, nil, nil)
    handler := NewDashboardHandler(dashboardSvc, nil)
    router := gin.New()
    router.GET("/admin/dashboard/account-success-rate-trend", handler.GetAccountSuccessRateTrend)

    req := httptest.NewRequest(http.MethodGet, "/admin/dashboard/account-success-rate-trend?start_date=2026-05-10&end_date=2026-05-11&granularity=1h", nil)
    rec := httptest.NewRecorder()
    router.ServeHTTP(rec, req)

    require.Equal(t, http.StatusOK, rec.Code)
    require.Equal(t, "miss", rec.Header().Get("X-Snapshot-Cache"))
}
```

```go
func TestAccountHandler_GetBatchSuccessRates_EmptyIDsReturnsEmptyMap(t *testing.T) {
    gin.SetMode(gin.TestMode)
    handler := NewAccountHandler(nil, nil, nil, nil, nil, nil, &service.AccountUsageService{}, nil, nil, nil, nil, nil)
    router := gin.New()
    router.POST("/admin/accounts/success-rate/batch", handler.GetBatchSuccessRates)

    req := httptest.NewRequest(http.MethodPost, "/admin/accounts/success-rate/batch", strings.NewReader(`{"account_ids":[]}`))
    req.Header.Set("Content-Type", "application/json")
    rec := httptest.NewRecorder()
    router.ServeHTTP(rec, req)

    require.Equal(t, http.StatusOK, rec.Code)
    require.JSONEq(t, `{"code":0,"message":"success","data":{"stats":{}}}`, rec.Body.String())
}
```

- [x] **Step 2: Run the backend unit/API-contract tests and verify they fail**

Run:

```bash
cd backend && go test -tags=unit ./internal/handler/admin ./internal/server -run 'TestDashboardHandler_GetAccountSuccessRateTrend|TestAccountHandler_GetBatchSuccessRates|TestAPIContracts' -count=1
```

Expected: FAIL with missing methods/routes.

- [x] **Step 3: Implement aggregate-table readers in `usage_log_repo.go`**

```go
// backend/internal/repository/usage_log_repo.go
func (r *usageLogRepository) GetDashboardSuccessRateStats(ctx context.Context, todayStart, historyStart time.Time) (*usagestats.SuccessRateSummary, *usagestats.SuccessRateSummary, error) {
    const q = `
WITH history AS (
    SELECT
        COALESCE(SUM(success_count), 0) AS success_count,
        COALESCE(SUM(failed_count), 0) AS failed_count,
        COALESCE(SUM(request_count), 0) AS request_count
    FROM account_request_stats_10m
    WHERE bucket_start >= $1
),
today AS (
    SELECT
        COALESCE(SUM(success_count), 0) AS success_count,
        COALESCE(SUM(failed_count), 0) AS failed_count,
        COALESCE(SUM(request_count), 0) AS request_count
    FROM account_request_stats_10m
    WHERE bucket_start >= $2
)
SELECT
    h.success_count, h.failed_count, h.request_count,
    t.success_count, t.failed_count, t.request_count
FROM history h
CROSS JOIN today t
`
}

func successRatePtr(successCount, requestCount int64) *float64 {
    if requestCount == 0 {
        return nil
    }
    value := math.Round((float64(successCount)/float64(requestCount))*10000) / 100
    return &value
}
```

```go
func (r *usageLogRepository) GetAccountSuccessRateBatch(ctx context.Context, accountIDs []int64) (map[int64]*usagestats.SuccessRateSummary, error) {
    const q = `
SELECT
    account_id,
    COALESCE(SUM(success_count), 0) AS success_count,
    COALESCE(SUM(failed_count), 0) AS failed_count,
    COALESCE(SUM(request_count), 0) AS request_count
FROM account_request_stats_10m
WHERE account_id = ANY($1)
GROUP BY account_id
`
```

```go
func (r *usageLogRepository) GetAccountSuccessRateTrend(
    ctx context.Context,
    startTime, endTime time.Time,
    granularity string,
    userTZ string,
) (*usagestats.AccountSuccessRateTrendResponse, error) {
    const q = `
WITH filtered AS (
    SELECT
        CASE
            WHEN $3 = '10m' THEN s.bucket_start
            WHEN $3 = '1h' THEN date_trunc('hour', s.bucket_start AT TIME ZONE $4) AT TIME ZONE $4
            ELSE date_trunc('day', s.bucket_start AT TIME ZONE $4) AT TIME ZONE $4
        END AS grouped_bucket_start,
        s.account_id,
        a.name AS account_name,
        s.success_count,
        s.failed_count,
        s.request_count
    FROM account_request_stats_10m s
    JOIN accounts a ON a.id = s.account_id
    WHERE s.bucket_start >= $1
      AND s.bucket_start < $2
      AND a.deleted_at IS NULL
      AND a.status = 'active'
      AND a.schedulable = TRUE
),
per_account AS (
    SELECT
        grouped_bucket_start,
        account_id,
        account_name,
        SUM(success_count) AS success_count,
        SUM(failed_count) AS failed_count,
        SUM(request_count) AS request_count
    FROM filtered
    GROUP BY grouped_bucket_start, account_id, account_name
)
SELECT
    grouped_bucket_start,
    account_id,
    account_name,
    success_count,
    failed_count,
    request_count
FROM per_account
ORDER BY grouped_bucket_start ASC, account_id ASC
`
```

- [x] **Step 4: Thread the new reads through services and handlers**

```go
// backend/internal/service/dashboard_service.go
func (s *DashboardService) GetDashboardStats(ctx context.Context) (*usagestats.DashboardStats, error) {
    stats, err := s.refreshDashboardStats(ctx)
    if err != nil {
        return nil, fmt.Errorf("get dashboard stats: %w", err)
    }
    return stats, nil
}

func (s *DashboardService) refreshDashboardStats(ctx context.Context) (*usagestats.DashboardStats, error) {
    stats, err := s.fetchDashboardStats(ctx)
    if err != nil {
        return nil, err
    }
    today, history, err := s.usageRepo.GetDashboardSuccessRateStats(ctx, timezone.Today(), truncateToDayUTC(time.Now().UTC().AddDate(0, 0, -s.aggUsageDays)))
    if err != nil {
        return nil, err
    }
    stats.TodaySuccessCount = today.SuccessCount
    stats.TodayFailedCount = today.FailedCount
    stats.TodaySuccessRate = today.SuccessRate
    stats.HistorySuccessRate = history.SuccessRate
    s.applyAggregationStatus(ctx, stats)
    cacheCtx, cancel := s.cacheOperationContext()
    defer cancel()
    s.saveDashboardStatsCache(cacheCtx, stats)
    return stats, nil
}
```

```go
// backend/internal/handler/admin/dashboard_snapshot_v2_handler.go
resp.Stats = &dashboardSnapshotV2Stats{
    DashboardStats: *stats,
    Uptime:         int64(time.Since(h.startTime).Seconds()),
}
```

```go
// backend/internal/handler/admin/dashboard_handler.go
func (h *DashboardHandler) GetAccountSuccessRateTrend(c *gin.Context) {
    startTime, endTime := parseTimeRange(c)
    granularity := strings.TrimSpace(c.DefaultQuery("granularity", "1h"))
    if granularity != "10m" && granularity != "1h" && granularity != "1d" {
        response.BadRequest(c, "Invalid granularity")
        return
    }

    data, err := h.dashboardService.GetAccountSuccessRateTrend(c.Request.Context(), startTime, endTime, granularity, c.Query("timezone"))
    if err != nil {
        response.Error(c, http.StatusInternalServerError, "Failed to get account success rate trend")
        return
    }
    response.Success(c, data)
}
```

```go
// backend/internal/handler/admin/account_handler.go
type BatchSuccessRateRequest struct {
    AccountIDs []int64 `json:"account_ids" binding:"required"`
}

func (h *AccountHandler) GetBatchSuccessRates(c *gin.Context) {
    var req BatchSuccessRateRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.BadRequest(c, "Invalid request: "+err.Error())
        return
    }

    accountIDs := normalizeInt64IDList(req.AccountIDs)
    if len(accountIDs) == 0 {
        response.Success(c, gin.H{"stats": map[string]any{}})
        return
    }

    cacheKey := buildAccountSuccessRateBatchCacheKey(accountIDs)
    if cached, ok := accountSuccessRateBatchCache.Get(cacheKey); ok {
        c.Header("X-Snapshot-Cache", "hit")
        response.Success(c, cached.Payload)
        return
    }

    stats, err := h.accountUsageService.GetHistorySuccessRateBatch(c.Request.Context(), accountIDs)
    if err != nil {
        response.ErrorFrom(c, err)
        return
    }
    payload := gin.H{"stats": stats}
    accountSuccessRateBatchCache.Set(cacheKey, payload)
    c.Header("X-Snapshot-Cache", "miss")
    response.Success(c, payload)
}
```

- [x] **Step 5: Register routes and contract coverage**

```go
// backend/internal/server/routes/admin.go
dashboard.GET("/account-success-rate-trend", h.Admin.Dashboard.GetAccountSuccessRateTrend)
accounts.POST("/success-rate/batch", h.Admin.Account.GetBatchSuccessRates)
```

```go
// backend/internal/server/api_contract_test.go
func (r *stubUsageLogRepo) GetDashboardSuccessRateStats(ctx context.Context, todayStart, historyStart time.Time) (*usagestats.SuccessRateSummary, *usagestats.SuccessRateSummary, error) {
    todayRate := 97.5
    historyRate := 99.1
    return &usagestats.SuccessRateSummary{SuccessCount: 39, FailedCount: 1, RequestCount: 40, SuccessRate: &todayRate},
        &usagestats.SuccessRateSummary{SuccessCount: 991, FailedCount: 9, RequestCount: 1000, SuccessRate: &historyRate},
        nil
}
```

```json
{
  "today_success_count": 39,
  "today_failed_count": 1,
  "today_success_rate": 97.5,
  "history_success_rate": 99.1
}
```

- [x] **Step 6: Run backend tests and commit**

Run:

```bash
cd backend && go test -tags=unit ./internal/handler/admin ./internal/server -run 'TestDashboardHandler_GetAccountSuccessRateTrend|TestAccountHandler_GetBatchSuccessRates|TestAPIContracts' -count=1
cd backend && go test -tags=integration ./internal/repository -run 'TestAccountRequestStats10m_|TestUsageLogRepoSuite/TestGetAccountWindowStats' -count=1
```

Expected: PASS.

- [x] **Step 7: Commit**

```bash
git add backend/internal/repository/usage_log_repo.go \
        backend/internal/service/dashboard_service.go \
        backend/internal/service/account_usage_service.go \
        backend/internal/handler/admin/dashboard_snapshot_v2_handler.go \
        backend/internal/handler/admin/dashboard_handler.go \
        backend/internal/handler/admin/account_handler.go \
        backend/internal/handler/admin/account_success_rate_cache.go \
        backend/internal/handler/admin/dashboard_success_rate_handler_test.go \
        backend/internal/handler/admin/id_list_utils_test.go \
        backend/internal/server/routes/admin.go \
        backend/internal/server/api_contract_test.go
git commit -m "feat: expose admin account success rate APIs"
```

### Task 4: Add Dashboard Success-Rate UI

**Files:**
- Modify: `frontend/src/types/index.ts`
- Modify: `frontend/src/api/admin/dashboard.ts`
- Modify: `frontend/src/views/admin/DashboardView.vue`
- Create: `frontend/src/components/charts/AccountSuccessRateTrend.vue`
- Create: `frontend/src/components/charts/__tests__/AccountSuccessRateTrend.spec.ts`
- Modify: `frontend/src/views/admin/__tests__/DashboardView.spec.ts`
- Modify: `frontend/src/i18n/locales/en.ts`
- Modify: `frontend/src/i18n/locales/zh.ts`

- [x] **Step 1: Write the failing dashboard/component tests**

```ts
// frontend/src/views/admin/__tests__/DashboardView.spec.ts
it('loads account success rate trend with default 1h granularity', async () => {
  getAccountSuccessRateTrend.mockResolvedValue({
    bucket: '1h',
    computed_at: '2026-05-11T10:00:00Z',
    stale: false,
    partial: false,
    points: []
  })

  mount(DashboardView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        DateRangePicker: true,
        Select: true,
        AccountSuccessRateTrend: true,
        ModelDistributionChart: true,
        TokenUsageTrend: true,
        Icon: true,
        LoadingSpinner: true
      }
    }
  })

  await flushPromises()

  expect(getAccountSuccessRateTrend).toHaveBeenCalledWith(expect.objectContaining({
    granularity: '1h'
  }))
})
```

```ts
// frontend/src/components/charts/__tests__/AccountSuccessRateTrend.spec.ts
it('renders empty state when no points are returned', () => {
  const wrapper = mount(AccountSuccessRateTrend, {
    props: {
      loading: false,
      trendData: {
        bucket: '1h',
        computed_at: '2026-05-11T10:00:00Z',
        stale: false,
        partial: false,
        points: []
      }
    }
  })

  expect(wrapper.text()).toContain('admin.dashboard.noDataAvailable')
})
```

- [x] **Step 2: Run the frontend tests and verify they fail**

Run:

```bash
cd frontend && pnpm test:run src/views/admin/__tests__/DashboardView.spec.ts src/components/charts/__tests__/AccountSuccessRateTrend.spec.ts
```

Expected: FAIL because the new API client method/component does not exist.

- [x] **Step 3: Extend shared types and API client**

```ts
// frontend/src/types/index.ts
export interface SuccessRateSummary {
  success_count: number
  failed_count: number
  request_count: number
  success_rate: number | null
}

export interface AccountSuccessRateTrendAccount extends SuccessRateSummary {
  account_id: number
  account_name: string
  success_rate: number
}

export interface AccountSuccessRateTrendPoint extends SuccessRateSummary {
  bucket_start: string
  success_rate: number
  accounts: AccountSuccessRateTrendAccount[]
}

export interface AccountSuccessRateTrendResponse {
  bucket: '10m' | '1h' | '1d'
  computed_at: string
  stale: boolean
  partial: boolean
  points: AccountSuccessRateTrendPoint[]
}
```

```ts
// frontend/src/api/admin/dashboard.ts
export interface AccountSuccessRateTrendParams {
  start_date?: string
  end_date?: string
  granularity?: '10m' | '1h' | '1d'
}

export async function getAccountSuccessRateTrend(
  params?: AccountSuccessRateTrendParams
): Promise<AccountSuccessRateTrendResponse> {
  const { data } = await apiClient.get<AccountSuccessRateTrendResponse>('/admin/dashboard/account-success-rate-trend', { params })
  return data
}
```

- [x] **Step 4: Replace the first dashboard card and add the new chart**

```vue
<!-- frontend/src/views/admin/DashboardView.vue -->
<div class="card p-4">
  <div class="flex items-center gap-3">
    <div class="rounded-lg bg-teal-100 p-2 dark:bg-teal-900/30">
      <Icon name="shield" size="md" class="text-teal-600 dark:text-teal-400" :stroke-width="2" />
    </div>
    <div>
      <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
        {{ t('admin.dashboard.requestSuccessRate') }}
      </p>
      <p class="text-xl font-bold text-gray-900 dark:text-white">
        {{ formatPercent(stats.today_success_rate) }}
      </p>
      <p class="text-xs text-gray-500 dark:text-gray-400">
        {{ t('admin.dashboard.historySuccessRate') }}: {{ formatPercent(stats.history_success_rate) }}
      </p>
    </div>
  </div>
</div>
```

```vue
<AccountSuccessRateTrend
  :trend-data="successRateTrend"
  :loading="successRateTrendLoading"
  :granularity="successRateGranularity"
  @refresh="loadSuccessRateTrend"
  @update:granularity="onSuccessRateGranularityChange"
/>
```

```ts
const successRateGranularity = ref<'10m' | '1h' | '1d'>('1h')
const successRateTrend = ref<AccountSuccessRateTrendResponse | null>(null)

const loadSuccessRateTrend = async () => {
  successRateTrendLoading.value = true
  try {
    successRateTrend.value = await adminAPI.dashboard.getAccountSuccessRateTrend({
      start_date: startDate.value,
      end_date: endDate.value,
      granularity: successRateGranularity.value
    })
  } finally {
    successRateTrendLoading.value = false
  }
}
```

- [x] **Step 5: Add translations and re-run frontend checks**

```ts
// frontend/src/i18n/locales/en.ts
requestSuccessRate: 'Request Success Rate',
historySuccessRate: 'History Success Rate',
todaySuccessRate: 'Today Success Rate',
successRateTrend: 'Account Success Rate Trend',
successRateGranularity10m: '10m',
successRateGranularity1h: '1h',
successRateGranularity1d: '1d',
```

```ts
// frontend/src/i18n/locales/zh.ts
requestSuccessRate: '请求成功率',
historySuccessRate: '历史成功率',
todaySuccessRate: '今日成功率',
successRateTrend: '账号请求成功率趋势',
successRateGranularity10m: '10 分钟',
successRateGranularity1h: '1 小时',
successRateGranularity1d: '1 天',
```

Run:

```bash
cd frontend && pnpm test:run src/views/admin/__tests__/DashboardView.spec.ts src/components/charts/__tests__/AccountSuccessRateTrend.spec.ts
cd frontend && pnpm typecheck
```

Expected: PASS.

- [x] **Step 6: Commit**

```bash
git add frontend/src/types/index.ts \
        frontend/src/api/admin/dashboard.ts \
        frontend/src/views/admin/DashboardView.vue \
        frontend/src/components/charts/AccountSuccessRateTrend.vue \
        frontend/src/components/charts/__tests__/AccountSuccessRateTrend.spec.ts \
        frontend/src/views/admin/__tests__/DashboardView.spec.ts \
        frontend/src/i18n/locales/en.ts \
        frontend/src/i18n/locales/zh.ts
git commit -m "feat: show dashboard account success rates"
```

### Task 5: Add The Account-List Success-Rate Column

**Files:**
- Modify: `frontend/src/api/admin/accounts.ts`
- Modify: `frontend/src/types/index.ts`
- Modify: `frontend/src/views/admin/AccountsView.vue`
- Create: `frontend/src/components/account/AccountSuccessRateCell.vue`
- Create: `frontend/src/components/account/__tests__/AccountSuccessRateCell.spec.ts`
- Create: `frontend/src/views/admin/__tests__/AccountsView.successRate.spec.ts`
- Modify: `frontend/src/components/account/index.ts`
- Modify: `frontend/src/i18n/locales/en.ts`
- Modify: `frontend/src/i18n/locales/zh.ts`

- [x] **Step 1: Write the failing accounts-page tests**

```ts
// frontend/src/views/admin/__tests__/AccountsView.successRate.spec.ts
it('fetches current-page success rates without blocking the account table', async () => {
  listAccounts.mockResolvedValue({
    items: [{ id: 101, name: 'claude-01', platform: 'openai', type: 'apikey', status: 'active', schedulable: true }],
    total: 1,
    page: 1,
    page_size: 20,
    pages: 1
  })
  getBatchSuccessRates.mockRejectedValueOnce(new Error('boom'))

  mount(AccountsView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        TablePageLayout: {
          template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>'
        },
        DataTable: { props: ['columns', 'data'], template: '<div data-test="data-table"></div>' },
        Pagination: true,
        ConfirmDialog: true,
        AccountTableActions: { template: '<div><slot name="beforeCreate" /><slot name="after" /></div>' },
        AccountTableFilters: { template: '<div></div>' },
        AccountBulkActionsBar: true,
        AccountActionMenu: true,
        ImportDataModal: true,
        ReAuthAccountModal: true,
        AccountTestModal: true,
        AccountStatsModal: true,
        ScheduledTestsPanel: true,
        SyncFromCrsModal: true,
        TempUnschedStatusModal: true,
        ErrorPassthroughRulesModal: true,
        TLSFingerprintProfilesModal: true,
        CreateAccountModal: true,
        EditAccountModal: true,
        BulkEditAccountModal: true,
        PlatformTypeBadge: true,
        AccountCapacityCell: true,
        AccountStatusIndicator: true,
        AccountTodayStatsCell: true,
        AccountSuccessRateCell: true,
        AccountGroupsCell: true,
        AccountUsageCell: true,
        Icon: true
      }
    }
  })
  await flushPromises()

  expect(listAccounts).toHaveBeenCalled()
  expect(getBatchSuccessRates).toHaveBeenCalledWith([101])
})
```

```ts
// frontend/src/components/account/__tests__/AccountSuccessRateCell.spec.ts
it('renders -- when request_count is zero', () => {
  const wrapper = mount(AccountSuccessRateCell, {
    props: {
      stats: { success_count: 0, failed_count: 0, request_count: 0, success_rate: null }
    }
  })

  expect(wrapper.text()).toContain('--')
})
```

- [x] **Step 2: Run the frontend tests and verify they fail**

Run:

```bash
cd frontend && pnpm test:run src/views/admin/__tests__/AccountsView.successRate.spec.ts src/components/account/__tests__/AccountSuccessRateCell.spec.ts
```

Expected: FAIL because the batch API and cell do not exist.

- [x] **Step 3: Add the batch API client and success-rate cell**

```ts
// frontend/src/api/admin/accounts.ts
export interface BatchSuccessRateResponse {
  stats: Record<string, SuccessRateSummary>
}

export async function getBatchSuccessRates(accountIds: number[]): Promise<BatchSuccessRateResponse> {
  const { data } = await apiClient.post<BatchSuccessRateResponse>('/admin/accounts/success-rate/batch', {
    account_ids: accountIds
  })
  return data
}
```

```vue
<!-- frontend/src/components/account/AccountSuccessRateCell.vue -->
<template>
  <div v-if="loading && !stats" class="space-y-1">
    <div class="h-3 w-14 animate-pulse rounded bg-gray-200 dark:bg-gray-700" />
    <div class="h-3 w-20 animate-pulse rounded bg-gray-200 dark:bg-gray-700" />
  </div>
  <div v-else-if="error && !stats" class="text-xs text-gray-400">--</div>
  <div v-else-if="stats" class="space-y-0.5 text-xs">
    <div class="font-medium text-gray-900 dark:text-white">
      {{ stats.success_rate == null ? '--' : `${stats.success_rate.toFixed(2)}%` }}
    </div>
    <div class="text-gray-500 dark:text-gray-400">
      {{ formatCount(stats.success_count) }} / {{ formatCount(stats.failed_count) }}
    </div>
  </div>
  <div v-else class="text-xs text-gray-400">--</div>
</template>
```

- [x] **Step 4: Wire the accounts page to batch-load the new column non-blockingly**

```ts
// frontend/src/views/admin/AccountsView.vue
const successRateByAccountId = ref<Record<string, SuccessRateSummary>>({})
const successRateLoading = ref(false)
const successRateError = ref<string | null>(null)

const refreshSuccessRatesBatch = async () => {
  if (hiddenColumns.has('success_rate')) {
    successRateLoading.value = false
    successRateError.value = null
    return
  }

  const accountIDs = accounts.value.map(account => account.id)
  if (accountIDs.length === 0) {
    successRateByAccountId.value = {}
    return
  }

  successRateLoading.value = true
  successRateError.value = null
  try {
    const result = await adminAPI.accounts.getBatchSuccessRates(accountIDs)
    const nextStats: Record<string, SuccessRateSummary> = {}
    for (const accountID of accountIDs) {
      nextStats[String(accountID)] = result.stats?.[String(accountID)] ?? {
        success_count: 0,
        failed_count: 0,
        request_count: 0,
        success_rate: null
      }
    }
    successRateByAccountId.value = nextStats
  } catch (error) {
    successRateError.value = 'Failed'
    console.error('Failed to load account success rates:', error)
  } finally {
    successRateLoading.value = false
  }
}
```

```ts
// put the column after today_stats
{ key: 'success_rate', label: t('admin.accounts.columns.successRate'), sortable: false }
```

```vue
<template #cell-success_rate="{ row }">
  <AccountSuccessRateCell
    :stats="successRateByAccountId[String(row.id)] ?? null"
    :loading="successRateLoading"
    :error="successRateError"
  />
</template>
```

```ts
// call alongside today stats after page loads, refreshes, and pagination/sort changes
await Promise.all([refreshTodayStatsBatch(), refreshSuccessRatesBatch()])
```

- [x] **Step 5: Add copy, exports, and re-run frontend verification**

```ts
// frontend/src/components/account/index.ts
export { default as AccountSuccessRateCell } from './AccountSuccessRateCell.vue'
```

```ts
// frontend/src/i18n/locales/en.ts
successRate: 'Success Rate',
```

```ts
// frontend/src/i18n/locales/zh.ts
successRate: '成功率',
```

Run:

```bash
cd frontend && pnpm test:run src/views/admin/__tests__/AccountsView.successRate.spec.ts src/components/account/__tests__/AccountSuccessRateCell.spec.ts
cd frontend && pnpm typecheck
```

Expected: PASS.

- [x] **Step 6: Commit**

```bash
git add frontend/src/api/admin/accounts.ts \
        frontend/src/views/admin/AccountsView.vue \
        frontend/src/components/account/AccountSuccessRateCell.vue \
        frontend/src/components/account/__tests__/AccountSuccessRateCell.spec.ts \
        frontend/src/views/admin/__tests__/AccountsView.successRate.spec.ts \
        frontend/src/components/account/index.ts \
        frontend/src/i18n/locales/en.ts \
        frontend/src/i18n/locales/zh.ts
git commit -m "feat: show account success rate column"
```

### Task 6: Final Verification

**Files:**
- Test: backend + frontend targeted suites

- [ ] **Step 1: Run backend unit tests**

Run:

```bash
cd backend && go test -tags=unit ./internal/handler/admin ./internal/service ./internal/server -count=1
```

Expected: PASS.

- [ ] **Step 2: Run backend integration tests for the new aggregate table**

Run:

```bash
cd backend && go test -tags=integration ./internal/repository -run 'TestMigrationsRunner_IsIdempotent_AndSchemaIsUpToDate|TestAccountRequestStats10m_' -count=1
```

Expected: PASS.

- [ ] **Step 3: Run frontend targeted tests**

Run:

```bash
cd frontend && pnpm test:run \
  src/views/admin/__tests__/DashboardView.spec.ts \
  src/views/admin/__tests__/AccountsView.successRate.spec.ts \
  src/components/charts/__tests__/AccountSuccessRateTrend.spec.ts \
  src/components/account/__tests__/AccountSuccessRateCell.spec.ts
```

Expected: PASS.

- [ ] **Step 4: Run frontend static checks**

Run:

```bash
cd frontend && pnpm lint:check && pnpm typecheck
```

Expected: PASS.

- [ ] **Step 5: Run cross-stack verification**

Run:

```bash
make test
```

Expected: PASS, or a narrow pre-existing failure that is unrelated to the success-rate changes and documented in the final handoff.

## Self-Review

**1. Spec coverage**

- Dashboard snapshot new fields: covered in Task 3.
- New `account-success-rate-trend` endpoint with `10m | 1h | 1d`: covered in Task 3 and Task 4.
- Account-list batch success-rate endpoint: covered in Task 3 and Task 5.
- Aggregate table + scheduler reuse + retention cleanup: covered in Task 1 and Task 2.
- Raw success/failure semantics, current-normal trend filtering, and no raw-log scans on read path: covered in Task 2 and Task 3.
- Dashboard/UI empty states and degraded list behavior: covered in Task 4 and Task 5.
- Backend/frontend tests from the spec: covered in Tasks 2 through 6.

**2. Placeholder scan**

- No `TODO`, `TBD`, or “implement later” text remains.
- Every task includes exact file paths, concrete commands, and representative code snippets.
- Failure expectations are explicit for each TDD-first test step.

**3. Type consistency**

- Backend and frontend both use `success_count`, `failed_count`, `request_count`, and `success_rate`.
- The dashboard trend granularity is consistently `10m | 1h | 1d`.
- The account-list batch route is consistently `/api/v1/admin/accounts/success-rate/batch`.

Plan complete and saved to `docs/superpowers/plans/2026-05-11-account-success-rate-implementation.md`. Two execution options:

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

**Which approach?**
