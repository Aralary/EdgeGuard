-- +goose Up
ALTER TABLE routes
    ADD COLUMN rate_limit_enabled BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN rate_limit_requests INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN rate_limit_window_seconds INTEGER NOT NULL DEFAULT 0,
    ADD CONSTRAINT routes_rate_limit_policy_check CHECK (
        (
            rate_limit_enabled = false
            AND rate_limit_requests = 0
            AND rate_limit_window_seconds = 0
        )
        OR
        (
            rate_limit_enabled = true
            AND rate_limit_requests > 0
            AND rate_limit_window_seconds > 0
        )
    );

-- +goose Down
ALTER TABLE routes
    DROP CONSTRAINT IF EXISTS routes_rate_limit_policy_check,
    DROP COLUMN IF EXISTS rate_limit_window_seconds,
    DROP COLUMN IF EXISTS rate_limit_requests,
    DROP COLUMN IF EXISTS rate_limit_enabled;
