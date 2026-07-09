package httpdelivery

import (
	"github.com/aralary/edgeguard/internal/demo/usecase"
	"github.com/aralary/edgeguard/internal/platform/logger"
)

type Handler struct {
	listOrders  *usecase.ListOrdersUseCase
	getOrder    *usecase.GetOrderUseCase
	createOrder *usecase.CreateOrderUseCase
	log         logger.Logger
}

func NewHandler(
	listOrders *usecase.ListOrdersUseCase,
	getOrder *usecase.GetOrderUseCase,
	createOrder *usecase.CreateOrderUseCase,
	log logger.Logger,
) *Handler {
	return &Handler{
		listOrders:  listOrders,
		getOrder:    getOrder,
		createOrder: createOrder,
		log:         log,
	}
}
