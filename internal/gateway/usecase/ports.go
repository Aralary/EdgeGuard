package usecase

import (
	"context"

	"github.com/aralary/edgeguard/internal/gateway/domain"
	"github.com/aralary/edgeguard/internal/platform/events"
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

type RateLimiter interface {
	Allow(ctx context.Context, request domain.RateLimitRequest) (domain.RateLimitResult, error)
}

type AccessEventPublisher interface {
	PublishGatewayAccess(ctx context.Context, event events.GatewayAccessEvent) error
}
