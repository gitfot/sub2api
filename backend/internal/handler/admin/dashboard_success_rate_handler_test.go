//go:build unit

package admin

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type accountSuccessRateRepoProbe struct {
	service.UsageLogRepository
	trendCalls          atomic.Int32
	batchCalls          atomic.Int32
	lastTrendTZ         string
	lastTrendGranular   string
	lastTrendAccountID  int64
	lastBatchAccountIDs []int64
}

func requireEqual(t *testing.T, want, got any) {
	t.Helper()
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func requireContains(t *testing.T, body string, needle string) {
	t.Helper()
	if !strings.Contains(body, needle) {
		t.Fatalf("expected body to contain %q, got %s", needle, body)
	}
}

func (r *accountSuccessRateRepoProbe) GetAccountSuccessRateTrend(
	ctx context.Context,
	startTime, endTime time.Time,
	granularity string,
	userTZ string,
	accountID int64,
) (*usagestats.AccountSuccessRateTrendResponse, error) {
	r.trendCalls.Add(1)
	r.lastTrendTZ = userTZ
	r.lastTrendGranular = granularity
	r.lastTrendAccountID = accountID
	return &usagestats.AccountSuccessRateTrendResponse{
		Bucket:     granularity,
		ComputedAt: "2026-03-11T10:00:00Z",
		Points: []usagestats.AccountSuccessRateTrendPoint{
			{
				BucketStart:  "2026-03-01T00:00:00-08:00",
				SuccessCount: 9,
				FailedCount:  1,
				RequestCount: 10,
				SuccessRate:  90,
			},
		},
	}, nil
}

func (r *accountSuccessRateRepoProbe) GetAccountSuccessRateBatch(
	ctx context.Context,
	accountIDs []int64,
) (map[int64]*usagestats.SuccessRateSummary, error) {
	r.batchCalls.Add(1)
	r.lastBatchAccountIDs = append([]int64(nil), accountIDs...)
	return map[int64]*usagestats.SuccessRateSummary{}, nil
}

func TestDashboardHandler_GetAccountSuccessRateTrend_UsesCache(t *testing.T) {
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

	req1 := httptest.NewRequest(
		http.MethodGet,
		"/admin/dashboard/account-success-rate-trend?start_date=2026-03-01&end_date=2026-03-01&granularity=1h&timezone=America/Los_Angeles",
		nil,
	)
	rec1 := httptest.NewRecorder()
	router.ServeHTTP(rec1, req1)

	requireEqual(t, http.StatusOK, rec1.Code)
	requireEqual(t, "miss", rec1.Header().Get("X-Snapshot-Cache"))
	requireContains(t, rec1.Body.String(), "\"bucket\":\"1h\"")
	requireContains(t, rec1.Body.String(), "\"success_rate\":90")
	requireEqual(t, "America/Los_Angeles", repo.lastTrendTZ)
	requireEqual(t, "1h", repo.lastTrendGranular)

	req2 := httptest.NewRequest(
		http.MethodGet,
		"/admin/dashboard/account-success-rate-trend?start_date=2026-03-01&end_date=2026-03-01&granularity=1h&timezone=America/Los_Angeles",
		nil,
	)
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)

	requireEqual(t, http.StatusOK, rec2.Code)
	requireEqual(t, "hit", rec2.Header().Get("X-Snapshot-Cache"))
	requireEqual(t, int32(1), repo.trendCalls.Load())
}

func TestDashboardHandler_GetAccountSuccessRateTrend_AccountIDAffectsCacheKey(t *testing.T) {
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

	req1 := httptest.NewRequest(
		http.MethodGet,
		"/admin/dashboard/account-success-rate-trend?start_date=2026-03-01&end_date=2026-03-01&granularity=1h&account_id=42",
		nil,
	)
	rec1 := httptest.NewRecorder()
	router.ServeHTTP(rec1, req1)

	requireEqual(t, http.StatusOK, rec1.Code)
	requireEqual(t, "miss", rec1.Header().Get("X-Snapshot-Cache"))

	req2 := httptest.NewRequest(
		http.MethodGet,
		"/admin/dashboard/account-success-rate-trend?start_date=2026-03-01&end_date=2026-03-01&granularity=1h&account_id=7",
		nil,
	)
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)

	requireEqual(t, http.StatusOK, rec2.Code)
	requireEqual(t, "miss", rec2.Header().Get("X-Snapshot-Cache"))
	requireEqual(t, int32(2), repo.trendCalls.Load())
	requireEqual(t, int64(7), repo.lastTrendAccountID)
}

func TestAccountHandler_GetBatchSuccessRates_EmptyIDsShortCircuits(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &accountSuccessRateRepoProbe{}
	accountUsageSvc := service.NewAccountUsageService(nil, repo, nil, nil, nil, nil, nil, nil)
	handler := NewAccountHandler(nil, nil, nil, nil, nil, nil, accountUsageSvc, nil, nil, nil, nil, nil, nil)
	router := gin.New()
	router.POST("/admin/accounts/success-rate/batch", handler.GetBatchSuccessRates)

	req := httptest.NewRequest(
		http.MethodPost,
		"/admin/accounts/success-rate/batch",
		bytes.NewBufferString(`{"account_ids":[]}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	requireEqual(t, http.StatusOK, rec.Code)
	requireContains(t, rec.Body.String(), `"stats":{}`)
	requireEqual(t, int32(0), repo.batchCalls.Load())
	requireEqual(t, "", rec.Header().Get("X-Snapshot-Cache"))
}

func TestAccountHandler_GetBatchSuccessRates_UsesCacheAndNormalizesIDs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	accountSuccessRateBatchCache = newSnapshotCache(30 * time.Second)
	t.Cleanup(func() {
		accountSuccessRateBatchCache = newSnapshotCache(30 * time.Second)
	})

	repo := &accountSuccessRateRepoProbe{}
	accountUsageSvc := service.NewAccountUsageService(nil, repo, nil, nil, nil, nil, nil, nil)
	handler := NewAccountHandler(nil, nil, nil, nil, nil, nil, accountUsageSvc, nil, nil, nil, nil, nil, nil)
	router := gin.New()
	router.POST("/admin/accounts/success-rate/batch", handler.GetBatchSuccessRates)

	body := `{"account_ids":[3,1,3,2,1]}`

	req1 := httptest.NewRequest(http.MethodPost, "/admin/accounts/success-rate/batch", bytes.NewBufferString(body))
	req1.Header.Set("Content-Type", "application/json")
	rec1 := httptest.NewRecorder()
	router.ServeHTTP(rec1, req1)

	requireEqual(t, http.StatusOK, rec1.Code)
	requireContains(t, rec1.Body.String(), `"stats":{}`)
	requireEqual(t, []int64{1, 2, 3}, repo.lastBatchAccountIDs)
	requireEqual(t, int32(1), repo.batchCalls.Load())
	etag := rec1.Header().Get("ETag")
	requireEqual(t, false, etag == "")

	req2 := httptest.NewRequest(http.MethodPost, "/admin/accounts/success-rate/batch", bytes.NewBufferString(body))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("If-None-Match", etag)
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)

	requireEqual(t, http.StatusNotModified, rec2.Code)
	requireEqual(t, int32(1), repo.batchCalls.Load())
}
