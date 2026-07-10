package postgres

import (
	"context"
	"fmt"

	"github.com/aralary/edgeguard/internal/controlplane/domain"
)

const createServiceQuery = `
INSERT INTO services (project_id, name, upstream_url)
VALUES ($1::uuid, $2, $3)
RETURNING id::text, project_id::text, name, upstream_url, created_at, updated_at
`

func (r *Repository) CreateService(ctx context.Context, service domain.Service) (domain.Service, error) {
	var created domain.Service

	err := r.db.QueryRow(
		ctx,
		createServiceQuery,
		service.ProjectID,
		service.Name,
		service.UpstreamURL,
	).Scan(
		&created.ID,
		&created.ProjectID,
		&created.Name,
		&created.UpstreamURL,
		&created.CreatedAt,
		&created.UpdatedAt,
	)
	if err != nil {
		return domain.Service{}, fmt.Errorf("create service: %w", err)
	}

	return created, nil
}

const listServicesByProjectIDQuery = `
SELECT id::text, project_id::text, name, upstream_url, created_at, updated_at
FROM services
WHERE project_id = $1::uuid
ORDER BY created_at ASC, id ASC
`

func (r *Repository) ListServicesByProjectID(ctx context.Context, projectID string) ([]domain.Service, error) {
	rows, err := r.db.Query(ctx, listServicesByProjectIDQuery, projectID)
	if err != nil {
		return nil, fmt.Errorf("list services by project id: %w", err)
	}
	defer rows.Close()

	services := make([]domain.Service, 0)

	for rows.Next() {
		var service domain.Service

		if err := rows.Scan(
			&service.ID,
			&service.ProjectID,
			&service.Name,
			&service.UpstreamURL,
			&service.CreatedAt,
			&service.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan service: %w", err)
		}

		services = append(services, service)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate services: %w", err)
	}

	return services, nil
}
