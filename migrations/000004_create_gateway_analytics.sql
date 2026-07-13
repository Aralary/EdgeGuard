-- +goose Up
CREATE TABLE gateway_access_events (
    event_id TEXT PRIMARY KEY,
    schema_version INTEGER NOT NULL CHECK (schema_version > 0),
    occurred_at TIMESTAMPTZ NOT NULL,
    request_id TEXT NOT NULL,
    project_id UUID,
    route_name TEXT,
    route_path_prefix TEXT,
    method TEXT NOT NULL,
    request_path TEXT NOT NULL,
    status_code INTEGER NOT NULL CHECK (status_code BETWEEN 100 AND 599),
    duration_ms BIGINT NOT NULL CHECK (duration_ms >= 0),
    response_bytes BIGINT NOT NULL CHECK (response_bytes >= 0),
    client_type TEXT NOT NULL CHECK (client_type IN ('api_key', 'ip')),
    client_id TEXT NOT NULL,
    auth_required BOOLEAN NOT NULL,
    rate_limit_enabled BOOLEAN NOT NULL,
    rate_limited BOOLEAN NOT NULL,
    ingested_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_gateway_access_events_occurred_at
    ON gateway_access_events(occurred_at DESC);

CREATE INDEX idx_gateway_access_events_project_route_time
    ON gateway_access_events(project_id, route_name, occurred_at DESC)
    WHERE project_id IS NOT NULL;

CREATE INDEX idx_gateway_access_events_status_time
    ON gateway_access_events(status_code, occurred_at DESC);

CREATE TABLE gateway_route_stats_hourly (
    bucket_start TIMESTAMPTZ NOT NULL,
    project_id UUID NOT NULL,
    route_name TEXT NOT NULL,
    route_path_prefix TEXT NOT NULL,
    method TEXT NOT NULL,
    request_count BIGINT NOT NULL DEFAULT 0 CHECK (request_count >= 0),
    status_1xx_count BIGINT NOT NULL DEFAULT 0 CHECK (status_1xx_count >= 0),
    status_2xx_count BIGINT NOT NULL DEFAULT 0 CHECK (status_2xx_count >= 0),
    status_3xx_count BIGINT NOT NULL DEFAULT 0 CHECK (status_3xx_count >= 0),
    status_4xx_count BIGINT NOT NULL DEFAULT 0 CHECK (status_4xx_count >= 0),
    status_5xx_count BIGINT NOT NULL DEFAULT 0 CHECK (status_5xx_count >= 0),
    rate_limited_count BIGINT NOT NULL DEFAULT 0 CHECK (rate_limited_count >= 0),
    total_duration_ms BIGINT NOT NULL DEFAULT 0 CHECK (total_duration_ms >= 0),
    max_duration_ms BIGINT NOT NULL DEFAULT 0 CHECK (max_duration_ms >= 0),
    total_response_bytes BIGINT NOT NULL DEFAULT 0 CHECK (total_response_bytes >= 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (
        bucket_start,
        project_id,
        route_name,
        route_path_prefix,
        method
    )
);

CREATE INDEX idx_gateway_route_stats_hourly_project_time
    ON gateway_route_stats_hourly(project_id, bucket_start DESC);

-- +goose Down
DROP TABLE IF EXISTS gateway_route_stats_hourly;
DROP TABLE IF EXISTS gateway_access_events;
