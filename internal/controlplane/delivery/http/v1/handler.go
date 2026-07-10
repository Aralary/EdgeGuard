package httpdelivery

import "github.com/aralary/edgeguard/internal/platform/logger"

type Handler struct {
	usecase ControlPlaneUsecase
	log     logger.Logger
}

func NewHandler(usecase ControlPlaneUsecase, log logger.Logger) *Handler {
	return &Handler{
		usecase: usecase,
		log:     log,
	}
}
