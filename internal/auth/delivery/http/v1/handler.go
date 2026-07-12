package httpdelivery

import "github.com/aralary/edgeguard/internal/platform/logger"

type Handler struct {
	usecase AuthUsecase
	log     logger.Logger
}

func NewHandler(usecase AuthUsecase, log logger.Logger) *Handler {
	return &Handler{
		usecase: usecase,
		log:     log,
	}
}
