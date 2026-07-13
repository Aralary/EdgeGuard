package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aralary/edgeguard/internal/analytics/domain"
	"github.com/aralary/edgeguard/internal/platform/events"
	"github.com/jackc/pgx/v5"
)

const insertGatewayAccessEventQuery = `
INSERT INTO gateway_access_events (
    event_id,
    schema_version,
    occurred_at,
    request_id,
    project_id,
    route_name,
    route_path_prefix,
    method,
    request_path,
    status_code,
    duration_ms,
    response_bytes,
    client_type,
    client_id,
    auth_required,
    rate_limit_enabled,
    rate_limited
)
VALUES (
    $1,
    $2,
    $3,
    $4,
    NULLIF($5, '')::uuid,
    NULLIF($6, ''),
    NULLIF($7, ''),
    $8,
    $9,
    $10,
    $11,
    $12,
    $13,
    $14,
    $15,
    $16,
    $17
)
ON CONFLICT (event_id) DO NOTHING
RETURNING event_id
`

const upsertGatewayRouteStatsHourlyQuery = `
INSERT INTO gateway_route_stats_hourly (
    bucket_start,
    project_id,
    route_name,
    route_path_prefix,
    method,
    request_count,
    status_1xx_count,
    status_2xx_count,
    status_3xx_count,
    status_4xx_count,
    status_5xx_count,
    rate_limited_count,
    total_duration_ms,
    max_duration_ms,
    total_response_bytes
)
VALUES (
    $1,
    $2::uuid,
    $3,
    $4,
    $5,
    1,
    $6,
    $7,
    $8,
    $9,
    $10,
    $11,
    $12,
    $12,
    $13
)
ON CONFLICT (
    bucket_start,
    project_id,
    route_name,
    route_path_prefix,
    method
)
DO UPDATE SET
    request_count = gateway_route_stats_hourly.request_count + 1,
    status_1xx_count = gateway_route_stats_hourly.status_1xx_count + EXCLUDED.status_1xx_count,
    status_2xx_count = gateway_route_stats_hourly.status_2xx_count + EXCLUDED.status_2xx_count,
    status_3xx_count = gateway_route_stats_hourly.status_3xx_count + EXCLUDED.status_3xx_count,
    status_4xx_count = gateway_route_stats_hourly.status_4xx_count + EXCLUDED.status_4xx_count,
    status_5xx_count = gateway_route_stats_hourly.status_5xx_count + EXCLUDED.status_5xx_count,
    rate_limited_count = gateway_route_stats_hourly.rate_limited_count + EXCLUDED.rate_limited_count,
    total_duration_ms = gateway_route_stats_hourly.total_duration_ms + EXCLUDED.total_duration_ms,
    max_duration_ms = GREATEST(gateway_route_stats_hourly.max_duration_ms, EXCLUDED.max_duration_ms),
    total_response_bytes = gateway_route_stats_hourly.total_response_bytes + EXCLUDED.total_response_bytes,
    updated_at = now()
`

func (r *Repository) StoreGatewayAccess(
	ctx context.Context,
	event events.GatewayAccessEvent,
) (domain.IngestionResult, error) {
	if r == nil || r.pool == nil {
		return domain.IngestionResult{}, errors.New("postgres pool is required")
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return domain.IngestionResult{}, fmt.Errorf("begin gateway analytics transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	var insertedEventID string
	err = tx.QueryRow(
		ctx,
		insertGatewayAccessEventQuery,
		event.EventID,
		event.SchemaVersion,
		event.OccurredAt.UTC(),
		event.RequestID,
		strings.TrimSpace(event.ProjectID),
		strings.TrimSpace(event.RouteName),
		strings.TrimSpace(event.RoutePathPrefix),
		strings.ToUpper(strings.TrimSpace(event.Method)),
		event.RequestPath,
		event.StatusCode,
		event.DurationMS,
		event.ResponseBytes,
		event.ClientType,
		event.ClientID,
		event.AuthRequired,
		event.RateLimitEnabled,
		event.RateLimited,
	).Scan(&insertedEventID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.IngestionResult{Duplicate: true}, nil
	}
	if err != nil {
		return domain.IngestionResult{}, fmt.Errorf("insert gateway access event: %w", err)
	}

	aggregated := shouldAggregate(event)
	if aggregated {
		counts := statusClassCounts(event.StatusCode)
		rateLimitedCount := int64(0)
		if event.RateLimited {
			rateLimitedCount = 1
		}

		_, err = tx.Exec(
			ctx,
			upsertGatewayRouteStatsHourlyQuery,
			event.OccurredAt.UTC().Truncate(time.Hour),
			strings.TrimSpace(event.ProjectID),
			strings.TrimSpace(event.RouteName),
			strings.TrimSpace(event.RoutePathPrefix),
			strings.ToUpper(strings.TrimSpace(event.Method)),
			counts.status1xx,
			counts.status2xx,
			counts.status3xx,
			counts.status4xx,
			counts.status5xx,
			rateLimitedCount,
			event.DurationMS,
			event.ResponseBytes,
		)
		if err != nil {
			return domain.IngestionResult{}, fmt.Errorf("upsert gateway route hourly stats: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.IngestionResult{}, fmt.Errorf("commit gateway analytics transaction: %w", err)
	}

	return domain.IngestionResult{Aggregated: aggregated}, nil
}

func shouldAggregate(event events.GatewayAccessEvent) bool {
	return strings.TrimSpace(event.ProjectID) != "" &&
		strings.TrimSpace(event.RouteName) != "" &&
		strings.TrimSpace(event.RoutePathPrefix) != ""
}

type statusCounts struct {
	status1xx int64
	status2xx int64
	status3xx int64
	status4xx int64
	status5xx int64
}

func statusClassCounts(statusCode int) statusCounts {
	var counts statusCounts

	switch statusCode / 100 {
	case 1:
		counts.status1xx = 1
	case 2:
		counts.status2xx = 1
	case 3:
		counts.status3xx = 1
	case 4:
		counts.status4xx = 1
	case 5:
		counts.status5xx = 1
	}

	return counts
}
