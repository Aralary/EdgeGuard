package httpdelivery

import "github.com/sirupsen/logrus"

type Handler struct {
	resolveRoute RouteResolver
	proxy        UpstreamProxy
	log          *logrus.Logger
}

func NewHandler(resolveRoute RouteResolver, proxy UpstreamProxy, log *logrus.Logger) *Handler {
	return &Handler{
		resolveRoute: resolveRoute,
		proxy:        proxy,
		log:          log,
	}
}
