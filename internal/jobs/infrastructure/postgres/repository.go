package postgres

import (
	"context"

	"github.com/aralary/edgeguard/internal/jobs/usecase"
	"github.com/jackc/pgx/v5"
)

type DBTX interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Repository struct {
	db DBTX
}

func New(db DBTX) *Repository {
	return &Repository{db: db}
}

var _ usecase.JobRepository = (*Repository)(nil)
