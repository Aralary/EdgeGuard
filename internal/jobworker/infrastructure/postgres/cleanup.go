package postgres

import (
	"context"
	"fmt"

	platformjobs "github.com/aralary/edgeguard/internal/platform/jobs"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TokenCleaner struct {
	pool *pgxpool.Pool
}

func NewTokenCleaner(pool *pgxpool.Pool) *TokenCleaner {
	return &TokenCleaner{pool: pool}
}

func (cleaner *TokenCleaner) CleanupExpiredTokens(ctx context.Context, payload platformjobs.CleanupPayload) (int64, error) {
	const query = `
WITH candidates AS (
    SELECT id
    FROM refresh_tokens
    WHERE expires_at < $1
       OR (revoked_at IS NOT NULL AND revoked_at < $1)
    ORDER BY created_at
    LIMIT $2
)
DELETE FROM refresh_tokens AS token
USING candidates
WHERE token.id = candidates.id
`

	result, err := cleaner.pool.Exec(ctx, query, payload.Before, payload.BatchSize)
	if err != nil {
		return 0, fmt.Errorf("delete expired refresh tokens: %w", err)
	}

	return result.RowsAffected(), nil
}
