package postgres

import (
	"context"
	"fmt"

	"github.com/aralary/edgeguard/internal/controlplane/domain"
)

const createRouteQuery = `
INSERT INTO routes (
    service_id,
    name,
    path_prefix,
    strip_prefix,
    timeout_ms,
    enabled
)
VALUES ($1::uuid, $2, $3, $4, $5, $6)
RETURNING
    id::text,
    service_id::text,
    name,
    path_prefix,
    strip_prefix,
    timeout_ms,
    enabled,
    created_at,
    updated_at
`

func (r *Repository) CreateRoute(ctx context.Context, route domain.Route) (domain.Route, error) {
	var created domain.Route

	err := r.db.QueryRow(
		ctx,
		createRouteQuery,
		route.ServiceID,
		route.Name,
		route.PathPrefix,
		route.StripPrefix,
		route.TimeoutMS,
		route.Enabled,
	).Scan(
		&created.ID,
		&created.ServiceID,
		&created.Name,
		&created.PathPrefix,
		&created.StripPrefix,
		&created.TimeoutMS,
		&created.Enabled,
		&created.CreatedAt,
		&created.UpdatedAt,
	)
	if err != nil {
		return domain.Route{}, fmt.Errorf("create route: %w", err)
	}

	return created, nil
}

const listRoutesByServiceIDQuery = `
SELECT
    id::text,
    service_id::text,
    name,
    path_prefix,
    strip_prefix,
    timeout_ms,
    enabled,
    created_at,
    updated_at
FROM routes
WHERE service_id = $1::uuid
ORDER BY created_at ASC, id ASC
`

func (r *Repository) ListRoutesByServiceID(ctx context.Context, serviceID string) ([]domain.Route, error) {
	rows, err := r.db.Query(ctx, listRoutesByServiceIDQuery, serviceID)
	if err != nil {
		return nil, fmt.Errorf("list routes by service id: %w", err)
	}
	defer rows.Close()

	routes := make([]domain.Route, 0)

	for rows.Next() {
		var route domain.Route

		if err := rows.Scan(
			&route.ID,
			&route.ServiceID,
			&route.Name,
			&route.PathPrefix,
			&route.StripPrefix,
			&route.TimeoutMS,
			&route.Enabled,
			&route.CreatedAt,
			&route.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan route: %w", err)
		}

		routes = append(routes, route)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate routes: %w", err)
	}

	return routes, nil
}
