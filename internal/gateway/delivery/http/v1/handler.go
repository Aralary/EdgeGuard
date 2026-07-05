package httpdelivery

import "github.com/aralary/edgeguard/internal/platform/logger"

type Handler struct {
	resolveRoute RouteResolver
	proxy        UpstreamProxy
	log          logger.Logger
}

func NewHandler(resolveRoute RouteResolver, proxy UpstreamProxy, log logger.Logger) *Handler {
	return &Handler{
		resolveRoute: resolveRoute,
		proxy:        proxy,
		log:          log,
	}
}
