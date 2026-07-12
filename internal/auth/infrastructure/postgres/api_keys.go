package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aralary/edgeguard/internal/auth/domain"
	"github.com/aralary/edgeguard/internal/auth/usecase"
	"github.com/jackc/pgx/v5"
)

const apiKeyProjectNameConstraint = "api_keys_project_id_name_key"

const createAPIKeyQuery = `
INSERT INTO api_keys (
    project_id,
    name,
    key_prefix,
    key_hash,
    enabled,
    expires_at
)
VALUES ($1::uuid, $2, $3, $4, $5, $6)
RETURNING
    id::text,
    project_id::text,
    name,
    key_prefix,
    key_hash,
    enabled,
    expires_at,
    last_used_at,
    created_at,
    updated_at
`

func (r *Repository) CreateAPIKey(
	ctx context.Context,
	key domain.APIKey,
) (domain.APIKey, error) {
	created, err := scanAPIKey(r.db.QueryRow(
		ctx,
		createAPIKeyQuery,
		key.ProjectID,
		key.Name,
		key.KeyPrefix,
		key.KeyHash,
		key.Enabled,
		key.ExpiresAt,
	))
	if err != nil {
		if isUniqueConstraintViolation(err, apiKeyProjectNameConstraint) {
			return domain.APIKey{}, usecase.ErrAPIKeyNameAlreadyExists
		}
		if isForeignKeyViolation(err) {
			return domain.APIKey{}, usecase.ErrProjectNotFound
		}

		return domain.APIKey{}, fmt.Errorf("create api key: %w", err)
	}

	return created, nil
}

const listAPIKeysByProjectIDQuery = `
SELECT
    id::text,
    project_id::text,
    name,
    key_prefix,
    key_hash,
    enabled,
    expires_at,
    last_used_at,
    created_at,
    updated_at
FROM api_keys
WHERE project_id = $1::uuid
ORDER BY created_at DESC, id DESC
`

func (r *Repository) ListAPIKeysByProjectID(
	ctx context.Context,
	projectID string,
) ([]domain.APIKey, error) {
	rows, err := r.db.Query(ctx, listAPIKeysByProjectIDQuery, projectID)
	if err != nil {
		return nil, fmt.Errorf("list api keys by project id: %w", err)
	}
	defer rows.Close()

	keys := make([]domain.APIKey, 0)
	for rows.Next() {
		key, err := scanAPIKey(rows)
		if err != nil {
			return nil, fmt.Errorf("scan api key: %w", err)
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate api keys: %w", err)
	}

	return keys, nil
}

const getAPIKeyByPrefixQuery = `
SELECT
    id::text,
    project_id::text,
    name,
    key_prefix,
    key_hash,
    enabled,
    expires_at,
    last_used_at,
    created_at,
    updated_at
FROM api_keys
WHERE key_prefix = $1
`

func (r *Repository) GetAPIKeyByPrefix(
	ctx context.Context,
	keyPrefix string,
) (domain.APIKey, error) {
	key, err := scanAPIKey(r.db.QueryRow(ctx, getAPIKeyByPrefixQuery, keyPrefix))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.APIKey{}, usecase.ErrAPIKeyNotFound
		}

		return domain.APIKey{}, fmt.Errorf("get api key by prefix: %w", err)
	}

	return key, nil
}

const disableAPIKeyQuery = `
UPDATE api_keys
SET
    enabled = false,
    updated_at = $3
WHERE id = $1::uuid
  AND project_id = $2::uuid
RETURNING id::text
`

func (r *Repository) DisableAPIKey(
	ctx context.Context,
	projectID string,
	apiKeyID string,
	disabledAt time.Time,
) error {
	var id string
	if err := r.db.QueryRow(
		ctx,
		disableAPIKeyQuery,
		apiKeyID,
		projectID,
		disabledAt,
	).Scan(&id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return usecase.ErrAPIKeyNotFound
		}

		return fmt.Errorf("disable api key: %w", err)
	}

	return nil
}

const updateAPIKeyLastUsedAtQuery = `
UPDATE api_keys
SET last_used_at = $2
WHERE id = $1::uuid
  AND enabled = true
RETURNING id::text
`

func (r *Repository) UpdateAPIKeyLastUsedAt(
	ctx context.Context,
	apiKeyID string,
	usedAt time.Time,
) error {
	var id string
	if err := r.db.QueryRow(
		ctx,
		updateAPIKeyLastUsedAtQuery,
		apiKeyID,
		usedAt,
	).Scan(&id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return usecase.ErrAPIKeyNotFound
		}

		return fmt.Errorf("update api key last used at: %w", err)
	}

	return nil
}

func scanAPIKey(row pgx.Row) (domain.APIKey, error) {
	var key domain.APIKey

	if err := row.Scan(
		&key.ID,
		&key.ProjectID,
		&key.Name,
		&key.KeyPrefix,
		&key.KeyHash,
		&key.Enabled,
		&key.ExpiresAt,
		&key.LastUsedAt,
		&key.CreatedAt,
		&key.UpdatedAt,
	); err != nil {
		return domain.APIKey{}, err
	}

	return key, nil
}
