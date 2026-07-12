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

const createRefreshTokenQuery = `
INSERT INTO refresh_tokens (
    user_id,
    token_hash,
    expires_at
)
VALUES ($1::uuid, $2, $3)
RETURNING
    id::text,
    user_id::text,
    token_hash,
    expires_at,
    revoked_at,
    created_at
`

func (r *Repository) CreateRefreshToken(
	ctx context.Context,
	token domain.RefreshToken,
) (domain.RefreshToken, error) {
	created, err := insertRefreshToken(ctx, r.db, token)
	if err != nil {
		return domain.RefreshToken{}, fmt.Errorf("create refresh token: %w", err)
	}

	return created, nil
}

const getRefreshTokenByHashQuery = `
SELECT
    id::text,
    user_id::text,
    token_hash,
    expires_at,
    revoked_at,
    created_at
FROM refresh_tokens
WHERE token_hash = $1
`

func (r *Repository) GetRefreshTokenByHash(
	ctx context.Context,
	tokenHash string,
) (domain.RefreshToken, error) {
	var token domain.RefreshToken

	err := r.db.QueryRow(ctx, getRefreshTokenByHashQuery, tokenHash).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.RevokedAt,
		&token.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.RefreshToken{}, usecase.ErrRefreshTokenNotFound
		}

		return domain.RefreshToken{}, fmt.Errorf("get refresh token by hash: %w", err)
	}

	return token, nil
}

const revokeRefreshTokenByIDQuery = `
UPDATE refresh_tokens
SET revoked_at = $2
WHERE id = $1::uuid
  AND revoked_at IS NULL
  AND expires_at > $2
RETURNING id::text
`

func (r *Repository) RotateRefreshToken(
	ctx context.Context,
	currentTokenID string,
	replacement domain.RefreshToken,
	revokedAt time.Time,
) (domain.RefreshToken, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return domain.RefreshToken{}, fmt.Errorf("begin refresh token rotation: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var revokedTokenID string
	if err := tx.QueryRow(
		ctx,
		revokeRefreshTokenByIDQuery,
		currentTokenID,
		revokedAt,
	).Scan(&revokedTokenID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.RefreshToken{}, usecase.ErrInvalidRefreshToken
		}

		return domain.RefreshToken{}, fmt.Errorf("revoke current refresh token: %w", err)
	}

	created, err := insertRefreshToken(ctx, tx, replacement)
	if err != nil {
		return domain.RefreshToken{}, fmt.Errorf("create replacement refresh token: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.RefreshToken{}, fmt.Errorf("commit refresh token rotation: %w", err)
	}

	return created, nil
}

const revokeRefreshTokenByHashQuery = `
UPDATE refresh_tokens
SET revoked_at = COALESCE(revoked_at, $2)
WHERE token_hash = $1
RETURNING id::text
`

func (r *Repository) RevokeRefreshTokenByHash(
	ctx context.Context,
	tokenHash string,
	revokedAt time.Time,
) error {
	var tokenID string

	err := r.db.QueryRow(
		ctx,
		revokeRefreshTokenByHashQuery,
		tokenHash,
		revokedAt,
	).Scan(&tokenID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return usecase.ErrRefreshTokenNotFound
		}

		return fmt.Errorf("revoke refresh token by hash: %w", err)
	}

	return nil
}

func insertRefreshToken(
	ctx context.Context,
	db interface {
		QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	},
	token domain.RefreshToken,
) (domain.RefreshToken, error) {
	var created domain.RefreshToken

	err := db.QueryRow(
		ctx,
		createRefreshTokenQuery,
		token.UserID,
		token.TokenHash,
		token.ExpiresAt,
	).Scan(
		&created.ID,
		&created.UserID,
		&created.TokenHash,
		&created.ExpiresAt,
		&created.RevokedAt,
		&created.CreatedAt,
	)
	if err != nil {
		return domain.RefreshToken{}, err
	}

	return created, nil
}
