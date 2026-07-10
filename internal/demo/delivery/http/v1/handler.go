package httpdelivery

import (
	"context"

	"github.com/aralary/edgeguard/internal/demo/domain"
	"github.com/aralary/edgeguard/internal/platform/logger"
)

type OrderUsecase interface {
	ListOrders(ctx context.Context) ([]domain.Order, error)
	GetOrder(ctx context.Context, id string) (domain.Order, error)
	CreateOrder(ctx context.Context, order domain.Order) (domain.Order, error)
}

type Handler struct {
	usecase OrderUsecase
	log     logger.Logger
}

func NewHandler(usecase OrderUsecase, log logger.Logger) *Handler {
	return &Handler{
		usecase: usecase,
		log:     log,
	}
}
