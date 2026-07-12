package httpdelivery

import "github.com/aralary/edgeguard/internal/platform/logger"

type Handler struct {
	usecase GatewayUsecase
	proxy   UpstreamProxy
	log     logger.Logger
}

func NewHandler(usecase GatewayUsecase, proxy UpstreamProxy, log logger.Logger) *Handler {
	return &Handler{
		usecase: usecase,
		proxy:   proxy,
		log:     log,
	}
}
