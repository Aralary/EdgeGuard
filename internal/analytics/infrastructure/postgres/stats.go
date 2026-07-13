package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/aralary/edgeguard/internal/analytics/domain"
)

const getRouteStatsSummaryQuery = `
SELECT
    COALESCE(SUM(request_count), 0),
    COALESCE(SUM(status_1xx_count), 0),
    COALESCE(SUM(status_2xx_count), 0),
    COALESCE(SUM(status_3xx_count), 0),
    COALESCE(SUM(status_4xx_count), 0),
    COALESCE(SUM(status_5xx_count), 0),
    COALESCE(SUM(rate_limited_count), 0),
    CASE
        WHEN COALESCE(SUM(request_count), 0) = 0 THEN 0
        ELSE SUM(total_duration_ms)::double precision / SUM(request_count)
    END,
    COALESCE(MAX(max_duration_ms), 0),
    COALESCE(SUM(total_response_bytes), 0)
FROM gateway_route_stats_hourly
WHERE project_id = $1::uuid
  AND bucket_start >= $2
  AND bucket_start < $3
  AND ($4 = '' OR route_name = $4)
  AND ($5 = '' OR method = $5)
`

const listRouteStatsHourlyQuery = `
SELECT
    bucket_start,
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
    CASE
        WHEN request_count = 0 THEN 0
        ELSE total_duration_ms::double precision / request_count
    END,
    max_duration_ms,
    total_response_bytes
FROM gateway_route_stats_hourly
WHERE project_id = $1::uuid
  AND bucket_start >= $2
  AND bucket_start < $3
  AND ($4 = '' OR route_name = $4)
  AND ($5 = '' OR method = $5)
ORDER BY bucket_start ASC, route_name ASC, method ASC
LIMIT $6
`

func (r *Repository) GetRouteStatsSummary(
	ctx context.Context,
	filter domain.RouteStatsFilter,
) (domain.RouteStatsSummary, error) {
	if r == nil || r.pool == nil {
		return domain.RouteStatsSummary{}, errors.New("postgres pool is required")
	}

	var summary domain.RouteStatsSummary
	if err := r.pool.QueryRow(
		ctx,
		getRouteStatsSummaryQuery,
		filter.ProjectID,
		filter.From,
		filter.To,
		filter.RouteName,
		filter.Method,
	).Scan(
		&summary.RequestCount,
		&summary.Status1xxCount,
		&summary.Status2xxCount,
		&summary.Status3xxCount,
		&summary.Status4xxCount,
		&summary.Status5xxCount,
		&summary.RateLimitedCount,
		&summary.AverageDurationMS,
		&summary.MaxDurationMS,
		&summary.TotalResponseBytes,
	); err != nil {
		return domain.RouteStatsSummary{}, fmt.Errorf("query route stats summary: %w", err)
	}

	return summary, nil
}

func (r *Repository) ListRouteStatsHourly(
	ctx context.Context,
	filter domain.RouteStatsFilter,
	limit int,
) ([]domain.RouteStatsHourly, error) {
	if r == nil || r.pool == nil {
		return nil, errors.New("postgres pool is required")
	}

	rows, err := r.pool.Query(
		ctx,
		listRouteStatsHourlyQuery,
		filter.ProjectID,
		filter.From,
		filter.To,
		filter.RouteName,
		filter.Method,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query hourly route stats: %w", err)
	}
	defer rows.Close()

	items := make([]domain.RouteStatsHourly, 0)
	for rows.Next() {
		var item domain.RouteStatsHourly
		if err := rows.Scan(
			&item.BucketStart,
			&item.RouteName,
			&item.RoutePathPrefix,
			&item.Method,
			&item.RequestCount,
			&item.Status1xxCount,
			&item.Status2xxCount,
			&item.Status3xxCount,
			&item.Status4xxCount,
			&item.Status5xxCount,
			&item.RateLimitedCount,
			&item.AverageDurationMS,
			&item.MaxDurationMS,
			&item.TotalResponseBytes,
		); err != nil {
			return nil, fmt.Errorf("scan hourly route stats: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate hourly route stats: %w", err)
	}

	return items, nil
}
