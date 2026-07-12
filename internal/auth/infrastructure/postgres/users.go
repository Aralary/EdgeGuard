package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/aralary/edgeguard/internal/auth/domain"
	"github.com/aralary/edgeguard/internal/auth/usecase"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const uniqueViolationCode = "23505"

const createUserQuery = `
INSERT INTO users (
    email,
    password_hash,
    role,
    enabled
)
VALUES ($1, $2, $3, $4)
RETURNING
    id::text,
    email,
    password_hash,
    role,
    enabled,
    created_at,
    updated_at
`

func (r *Repository) CreateUser(ctx context.Context, user domain.User) (domain.User, error) {
	var created domain.User

	err := r.db.QueryRow(
		ctx,
		createUserQuery,
		user.Email,
		user.PasswordHash,
		user.Role,
		user.Enabled,
	).Scan(
		&created.ID,
		&created.Email,
		&created.PasswordHash,
		&created.Role,
		&created.Enabled,
		&created.CreatedAt,
		&created.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.User{}, usecase.ErrEmailAlreadyExists
		}

		return domain.User{}, fmt.Errorf("create user: %w", err)
	}

	return created, nil
}

const getUserByEmailQuery = `
SELECT
    id::text,
    email,
    password_hash,
    role,
    enabled,
    created_at,
    updated_at
FROM users
WHERE lower(email) = lower($1)
`

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (domain.User, error) {
	user, err := scanUser(r.db.QueryRow(ctx, getUserByEmailQuery, email))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, usecase.ErrUserNotFound
		}

		return domain.User{}, fmt.Errorf("get user by email: %w", err)
	}

	return user, nil
}

const getUserByIDQuery = `
SELECT
    id::text,
    email,
    password_hash,
    role,
    enabled,
    created_at,
    updated_at
FROM users
WHERE id = $1::uuid
`

func (r *Repository) GetUserByID(ctx context.Context, id string) (domain.User, error) {
	user, err := scanUser(r.db.QueryRow(ctx, getUserByIDQuery, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, usecase.ErrUserNotFound
		}

		return domain.User{}, fmt.Errorf("get user by id: %w", err)
	}

	return user, nil
}

func scanUser(row pgx.Row) (domain.User, error) {
	var user domain.User

	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.Enabled,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return domain.User{}, err
	}

	return user, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode
}
