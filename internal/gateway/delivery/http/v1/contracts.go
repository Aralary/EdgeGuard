package httpdelivery

import (
	"context"
	"net/http"

	"github.com/aralary/edgeguard/internal/gateway/domain"
)

type RouteResolver interface {
	ResolveRoute(ctx context.Context, path string) (domain.Route, error)
}

type UpstreamProxy interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request, route domain.Route)
}
