package usecase

import (
	"context"

	"github.com/aralary/edgeguard/internal/gateway/domain"
)

type RouteRepository interface {
	ListRoutes(ctx context.Context) ([]domain.Route, error)
	ReplaceRoutes(routes []domain.Route)
}

type RouteSource interface {
	ListRoutes(ctx context.Context) ([]domain.Route, error)
}

type APIKeyValidator interface {
	ValidateAPIKey(ctx context.Context, rawAPIKey string) (domain.APIKeyPrincipal, error)
}
