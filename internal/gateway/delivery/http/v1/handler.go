package httpdelivery

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/aralary/edgeguard/internal/gateway/domain"
)

type RouteResolver interface {
	Execute(ctx context.Context, path string) (domain.Route, error)
}

type UpstreamProxy interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request, route domain.Route)
}

type Handler struct {
	resolveRoute RouteResolver
	proxy        UpstreamProxy
	log          *slog.Logger
}

func NewHandler(resolveRoute RouteResolver, proxy UpstreamProxy, log *slog.Logger) *Handler {
	return &Handler{
		resolveRoute: resolveRoute,
		proxy:        proxy,
		log:          log,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	route, err := h.resolveRoute.Execute(r.Context(), r.URL.Path)
	if err != nil {
		if errors.Is(err, domain.ErrRouteNotFound) {
			http.Error(w, "route not found", http.StatusNotFound)
			return
		}

		h.log.Error("failed to resolve route", slog.String("error", err.Error()))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.proxy.ServeHTTP(w, r, route)
}