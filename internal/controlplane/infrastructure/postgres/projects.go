package postgres

import (
	"context"
	"fmt"

	"github.com/aralary/edgeguard/internal/controlplane/domain"
)

const createProjectQuery = `
INSERT INTO projects (name)
VALUES ($1)
RETURNING id::text, name, created_at, updated_at
`

func (r *Repository) CreateProject(ctx context.Context, project domain.Project) (domain.Project, error) {
	var created domain.Project

	err := r.db.QueryRow(ctx, createProjectQuery, project.Name).Scan(
		&created.ID,
		&created.Name,
		&created.CreatedAt,
		&created.UpdatedAt,
	)
	if err != nil {
		return domain.Project{}, fmt.Errorf("create project: %w", err)
	}

	return created, nil
}

const listProjectsQuery = `
SELECT id::text, name, created_at, updated_at
FROM projects
ORDER BY created_at ASC, id ASC
`

func (r *Repository) ListProjects(ctx context.Context) ([]domain.Project, error) {
	rows, err := r.db.Query(ctx, listProjectsQuery)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	projects := make([]domain.Project, 0)

	for rows.Next() {
		var project domain.Project

		if err := rows.Scan(
			&project.ID,
			&project.Name,
			&project.CreatedAt,
			&project.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}

		projects = append(projects, project)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate projects: %w", err)
	}

	return projects, nil
}
