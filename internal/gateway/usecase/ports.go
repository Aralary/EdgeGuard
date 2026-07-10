package usecase

import (
	"context"

	"github.com/aralary/edgeguard/internal/gateway/domain"
)

type RouteRepository interface {
	ListRoutes(ctx context.Context) ([]domain.Route, error)
}
