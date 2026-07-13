-- +goose Up
CREATE TABLE background_jobs (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    status TEXT NOT NULL,
    current_attempt INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    queued_at TIMESTAMPTZ,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL,
    last_error TEXT NOT NULL DEFAULT '',
    result_message TEXT NOT NULL DEFAULT '',
    output_path TEXT NOT NULL DEFAULT '',
    affected_rows BIGINT NOT NULL DEFAULT 0,
    CONSTRAINT background_jobs_status_check CHECK (
        status IN ('publishing', 'publish_failed', 'queued', 'processing', 'retrying', 'succeeded', 'failed')
    ),
    CONSTRAINT background_jobs_attempt_check CHECK (
        current_attempt >= 0 AND max_attempts >= 1 AND current_attempt <= max_attempts
    )
);

CREATE INDEX background_jobs_status_created_at_idx
    ON background_jobs (status, created_at DESC);

CREATE INDEX background_jobs_type_created_at_idx
    ON background_jobs (type, created_at DESC);

-- +goose Down
DROP TABLE background_jobs;
