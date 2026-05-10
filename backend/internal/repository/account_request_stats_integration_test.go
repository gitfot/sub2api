//go:build integration

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type accountRequestStatsRow struct {
	bucketStart  time.Time
	accountID    int64
	successCount int64
	failedCount  int64
	requestCount int64
}

func TestAccountRequestStats10m_AggregateRange(t *testing.T) {
	ctx := context.Background()
	tx := testEntTx(t)
	client := tx.Client()
	usageRepo := newUsageLogRepositoryWithSQL(client, tx)
	aggRepo := newDashboardAggregationRepositoryWithSQL(tx)

	account1 := mustCreateAccount(t, client, &service.Account{Name: "agg-10m-1"})
	account2 := mustCreateAccount(t, client, &service.Account{Name: "agg-10m-2"})
	user := mustCreateUser(t, client, &service.User{Email: fmt.Sprintf("agg-10m-%d@example.com", time.Now().UnixNano())})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{UserID: user.ID, Key: "sk-agg-10m", Name: "agg-10m"})

	logTimes := []time.Time{
		time.Date(2026, 5, 10, 10, 1, 0, 0, time.UTC),
		time.Date(2026, 5, 10, 10, 9, 59, 0, time.UTC),
		time.Date(2026, 5, 10, 10, 12, 0, 0, time.UTC),
	}
	for i, createdAt := range logTimes {
		_, err := usageRepo.Create(ctx, &service.UsageLog{
			UserID:          user.ID,
			APIKeyID:        apiKey.ID,
			AccountID:       account1.ID,
			RequestID:       uuid.NewString(),
			Model:           "claude-3",
			InputTokens:     10 + i,
			OutputTokens:    20 + i,
			TotalCost:       0.5,
			ActualCost:      0.5,
			InboundEndpoint: accountStatsStringPtr("/v1/messages"),
			CreatedAt:       createdAt,
		})
		require.NoError(t, err)
	}

	_, err := usageRepo.Create(ctx, &service.UsageLog{
		UserID:          user.ID,
		APIKeyID:        apiKey.ID,
		AccountID:       account1.ID,
		RequestID:       uuid.NewString(),
		Model:           "claude-3",
		InputTokens:     1,
		OutputTokens:    1,
		TotalCost:       0.1,
		ActualCost:      0.1,
		InboundEndpoint: accountStatsStringPtr("/v1/messages/count_tokens"),
		CreatedAt:       time.Date(2026, 5, 10, 10, 4, 0, 0, time.UTC),
	})
	require.NoError(t, err)

	_, err = usageRepo.Create(ctx, &service.UsageLog{
		UserID:          user.ID,
		APIKeyID:        apiKey.ID,
		AccountID:       account2.ID,
		RequestID:       uuid.NewString(),
		Model:           "claude-3",
		InputTokens:     5,
		OutputTokens:    6,
		TotalCost:       0.2,
		ActualCost:      0.2,
		InboundEndpoint: accountStatsStringPtr("/v1/messages"),
		CreatedAt:       time.Date(2026, 5, 10, 10, 5, 0, 0, time.UTC),
	})
	require.NoError(t, err)

	err = insertOpsErrorLogInTx(ctx, tx, &service.OpsInsertErrorLogInput{
		RequestID:       uuid.NewString(),
		AccountID:       &account1.ID,
		StatusCode:      500,
		Severity:        "error",
		IsCountTokens:   false,
		InboundEndpoint: "/v1/messages",
		ErrorPhase:      "upstream",
		ErrorType:       "provider_error",
		CreatedAt:       time.Date(2026, 5, 10, 10, 7, 0, 0, time.UTC),
	})
	require.NoError(t, err)

	err = insertOpsErrorLogInTx(ctx, tx, &service.OpsInsertErrorLogInput{
		RequestID:       uuid.NewString(),
		AccountID:       &account2.ID,
		StatusCode:      429,
		Severity:        "error",
		IsCountTokens:   false,
		InboundEndpoint: "/v1/messages",
		ErrorPhase:      "upstream",
		ErrorType:       "rate_limited",
		CreatedAt:       time.Date(2026, 5, 10, 10, 13, 0, 0, time.UTC),
	})
	require.NoError(t, err)

	err = insertOpsErrorLogInTx(ctx, tx, &service.OpsInsertErrorLogInput{
		RequestID:       uuid.NewString(),
		AccountID:       &account1.ID,
		StatusCode:      503,
		Severity:        "error",
		IsCountTokens:   true,
		InboundEndpoint: "/v1/messages/count_tokens",
		ErrorPhase:      "upstream",
		ErrorType:       "probe_error",
		CreatedAt:       time.Date(2026, 5, 10, 10, 8, 0, 0, time.UTC),
	})
	require.NoError(t, err)

	err = insertOpsErrorLogInTx(ctx, tx, &service.OpsInsertErrorLogInput{
		RequestID:       uuid.NewString(),
		StatusCode:      500,
		Severity:        "error",
		IsCountTokens:   false,
		InboundEndpoint: "/v1/messages",
		ErrorPhase:      "upstream",
		ErrorType:       "missing_account",
		CreatedAt:       time.Date(2026, 5, 10, 10, 6, 0, 0, time.UTC),
	})
	require.NoError(t, err)

	start := time.Date(2026, 5, 10, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 10, 10, 20, 0, 0, time.UTC)
	require.NoError(t, aggRepo.AggregateRange(ctx, start, end))

	rows := fetchAccountRequestStatsRows(t, tx, start, end)
	require.Equal(t, []accountRequestStatsRow{
		{
			bucketStart:  time.Date(2026, 5, 10, 10, 0, 0, 0, time.UTC),
			accountID:    account1.ID,
			successCount: 2,
			failedCount:  1,
			requestCount: 3,
		},
		{
			bucketStart:  time.Date(2026, 5, 10, 10, 0, 0, 0, time.UTC),
			accountID:    account2.ID,
			successCount: 1,
			failedCount:  0,
			requestCount: 1,
		},
		{
			bucketStart:  time.Date(2026, 5, 10, 10, 10, 0, 0, time.UTC),
			accountID:    account1.ID,
			successCount: 1,
			failedCount:  0,
			requestCount: 1,
		},
		{
			bucketStart:  time.Date(2026, 5, 10, 10, 10, 0, 0, time.UTC),
			accountID:    account2.ID,
			successCount: 0,
			failedCount:  1,
			requestCount: 1,
		},
	}, rows)
}

func TestAccountRequestStats10m_RecomputeRangeRebuildsRows(t *testing.T) {
	ctx := context.Background()
	tx := testEntTx(t)
	client := tx.Client()
	usageRepo := newUsageLogRepositoryWithSQL(client, tx)
	aggRepo := newDashboardAggregationRepositoryWithSQL(tx)

	account := mustCreateAccount(t, client, &service.Account{Name: "recompute-10m"})
	user := mustCreateUser(t, client, &service.User{Email: fmt.Sprintf("recompute-10m-%d@example.com", time.Now().UnixNano())})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{UserID: user.ID, Key: "sk-recompute-10m", Name: "recompute-10m"})

	requestID := uuid.NewString()
	createdAt := time.Date(2026, 5, 10, 11, 2, 0, 0, time.UTC)
	_, err := usageRepo.Create(ctx, &service.UsageLog{
		UserID:          user.ID,
		APIKeyID:        apiKey.ID,
		AccountID:       account.ID,
		RequestID:       requestID,
		Model:           "claude-3",
		InputTokens:     10,
		OutputTokens:    10,
		TotalCost:       0.3,
		ActualCost:      0.3,
		InboundEndpoint: accountStatsStringPtr("/v1/messages"),
		CreatedAt:       createdAt,
	})
	require.NoError(t, err)

	start := time.Date(2026, 5, 10, 11, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 10, 11, 10, 0, 0, time.UTC)
	require.NoError(t, aggRepo.AggregateRange(ctx, start, end))

	_, err = tx.ExecContext(ctx, `
		INSERT INTO account_request_stats_10m (bucket_start, account_id, success_count, failed_count, request_count, computed_at)
		VALUES ($1, $2, 99, 0, 99, NOW())
	`, start, mustCreateAccount(t, client, &service.Account{Name: "stale-10m"}).ID)
	require.NoError(t, err)

	_, err = tx.ExecContext(ctx, "DELETE FROM usage_logs WHERE request_id = $1", requestID)
	require.NoError(t, err)

	require.NoError(t, aggRepo.RecomputeRange(ctx, start, end))
	require.Empty(t, fetchAccountRequestStatsRows(t, tx, start, end))
}

func TestAccountRequestStats10m_CleanupAggregates(t *testing.T) {
	ctx := context.Background()
	tx := testEntTx(t)
	client := tx.Client()
	aggRepo := newDashboardAggregationRepositoryWithSQL(tx)
	account := mustCreateAccount(t, client, &service.Account{Name: "cleanup-10m"})

	oldBucket := time.Date(2026, 5, 8, 9, 50, 0, 0, time.UTC)
	newBucket := time.Date(2026, 5, 10, 9, 50, 0, 0, time.UTC)
	for _, bucket := range []time.Time{oldBucket, newBucket} {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO account_request_stats_10m (bucket_start, account_id, success_count, failed_count, request_count, computed_at)
			VALUES ($1, $2, 1, 0, 1, NOW())
		`, bucket, account.ID)
		require.NoError(t, err)
	}

	require.NoError(t, aggRepo.CleanupAggregates(
		ctx,
		time.Date(2026, 5, 9, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 9, 0, 0, 0, 0, time.UTC),
	))

	rows := fetchAccountRequestStatsRows(t, tx, time.Date(2026, 5, 8, 0, 0, 0, 0, time.UTC), time.Date(2026, 5, 11, 0, 0, 0, 0, time.UTC))
	require.Equal(t, []accountRequestStatsRow{
		{
			bucketStart:  newBucket,
			accountID:    account.ID,
			successCount: 1,
			failedCount:  0,
			requestCount: 1,
		},
	}, rows)
}

func fetchAccountRequestStatsRows(t *testing.T, db queryer, start, end time.Time) []accountRequestStatsRow {
	t.Helper()

	rows, err := db.QueryContext(context.Background(), `
		SELECT bucket_start, account_id, success_count, failed_count, request_count
		FROM account_request_stats_10m
		WHERE bucket_start >= $1 AND bucket_start < $2
		ORDER BY bucket_start, account_id
	`, start, end)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()

	var result []accountRequestStatsRow
	for rows.Next() {
		var row accountRequestStatsRow
		require.NoError(t, rows.Scan(&row.bucketStart, &row.accountID, &row.successCount, &row.failedCount, &row.requestCount))
		result = append(result, row)
	}
	require.NoError(t, rows.Err())
	return result
}

type queryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func accountStatsStringPtr(v string) *string {
	return &v
}

func insertOpsErrorLogInTx(ctx context.Context, tx sqlExecutor, input *service.OpsInsertErrorLogInput) error {
	_, err := tx.ExecContext(ctx, insertOpsErrorLogSQL, opsInsertErrorLogArgs(input)...)
	return err
}
