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

	mock.ExpectQuery(`(?s)upstream_status_code.+upstream_errors.+upstream_excl`).
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

	mock.ExpectQuery(`(?s)upstream_status_code.+upstream_errors.+upstream_excl`).
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
