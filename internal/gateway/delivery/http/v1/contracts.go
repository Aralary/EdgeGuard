package httpdelivery

import (
	"context"
	"net/http"

	"github.com/aralary/edgeguard/internal/gateway/domain"
)

type GatewayUsecase interface {
	ResolveRoute(ctx context.Context, path string) (domain.Route, error)
	AuthorizeRoute(ctx context.Context, route domain.Route, rawAPIKey string) error
}

type UpstreamProxy interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request, route domain.Route)
}
