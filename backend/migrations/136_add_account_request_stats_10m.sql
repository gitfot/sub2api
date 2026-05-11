CREATE TABLE IF NOT EXISTS account_request_stats_10m (
    bucket_start TIMESTAMPTZ NOT NULL,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    success_count BIGINT NOT NULL DEFAULT 0,
    failed_count BIGINT NOT NULL DEFAULT 0,
    request_count BIGINT NOT NULL DEFAULT 0,
    computed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (bucket_start, account_id),
    CONSTRAINT account_request_stats_10m_request_count_check
        CHECK (request_count = success_count + failed_count)
);

CREATE INDEX IF NOT EXISTS idx_account_request_stats_10m_account_bucket_desc
    ON account_request_stats_10m (account_id, bucket_start DESC);

CREATE INDEX IF NOT EXISTS idx_account_request_stats_10m_bucket_desc
    ON account_request_stats_10m (bucket_start DESC);

COMMENT ON TABLE account_request_stats_10m IS '10-minute account request success/failure aggregates.';
COMMENT ON COLUMN account_request_stats_10m.bucket_start IS 'UTC bucket start for the 10-minute aggregate.';
COMMENT ON COLUMN account_request_stats_10m.computed_at IS 'When the aggregate row was last computed.';
