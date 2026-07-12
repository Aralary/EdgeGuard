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
    enabled,
    auth_required
)
VALUES ($1::uuid, $2, $3, $4, $5, $6, $7)
RETURNING
    id::text,
    service_id::text,
    name,
    path_prefix,
    strip_prefix,
    timeout_ms,
    enabled,
    auth_required,
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
		route.AuthRequired,
	).Scan(
		&created.ID,
		&created.ServiceID,
		&created.Name,
		&created.PathPrefix,
		&created.StripPrefix,
		&created.TimeoutMS,
		&created.Enabled,
		&created.AuthRequired,
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
    auth_required,
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
			&route.AuthRequired,
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

const listGatewayRoutesQuery = `
SELECT
    services.project_id::text,
    routes.name,
    routes.path_prefix,
    services.upstream_url,
    routes.strip_prefix,
    routes.timeout_ms,
    routes.auth_required
FROM routes
JOIN services ON services.id = routes.service_id
WHERE routes.enabled = TRUE
ORDER BY length(routes.path_prefix) DESC, routes.path_prefix ASC, routes.id ASC
`

func (r *Repository) ListGatewayRoutes(ctx context.Context) ([]domain.GatewayRoute, error) {
	rows, err := r.db.Query(ctx, listGatewayRoutesQuery)
	if err != nil {
		return nil, fmt.Errorf("list gateway routes: %w", err)
	}
	defer rows.Close()

	routes := make([]domain.GatewayRoute, 0)

	for rows.Next() {
		var route domain.GatewayRoute

		if err := rows.Scan(
			&route.ProjectID,
			&route.Name,
			&route.PathPrefix,
			&route.UpstreamURL,
			&route.StripPrefix,
			&route.TimeoutMS,
			&route.AuthRequired,
		); err != nil {
			return nil, fmt.Errorf("scan gateway route: %w", err)
		}

		routes = append(routes, route)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate gateway routes: %w", err)
	}

	return routes, nil
}
