package postgres

import (
	"context"

	"github.com/aralary/edgeguard/internal/auth/usecase"
	"github.com/jackc/pgx/v5"
)

type DBTX interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
}

type Repository struct {
	db DBTX
}

func New(db DBTX) *Repository {
	return &Repository{db: db}
}

var _ usecase.UserRepository = (*Repository)(nil)
var _ usecase.RefreshTokenRepository = (*Repository)(nil)
