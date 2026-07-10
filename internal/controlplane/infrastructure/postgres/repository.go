package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type DBTX interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Repository struct {
	db DBTX
}

func New(db DBTX) *Repository {
	return &Repository{db: db}
}
