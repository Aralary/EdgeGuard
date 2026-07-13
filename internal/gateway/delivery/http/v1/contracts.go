package httpdelivery

import (
	"context"
	"net/http"

	"github.com/aralary/edgeguard/internal/gateway/domain"
	"github.com/aralary/edgeguard/internal/platform/events"
)

type AccessEventUsecase interface {
	PublishAccessEvent(ctx context.Context, event events.GatewayAccessEvent) error
}

type GatewayUsecase interface {
	AccessEventUsecase
	ResolveRoute(ctx context.Context, path string) (domain.Route, error)
	AuthorizeRoute(
		ctx context.Context,
		route domain.Route,
		rawAPIKey string,
	) (domain.APIKeyPrincipal, error)
	CheckRateLimit(
		ctx context.Context,
		route domain.Route,
		clientID string,
	) (domain.RateLimitResult, error)
}

type UpstreamProxy interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request, route domain.Route)
}
