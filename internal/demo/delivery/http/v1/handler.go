package httpdelivery

import (
	"github.com/aralary/edgeguard/internal/demo/usecase"
	"github.com/sirupsen/logrus"
)

type Handler struct {
	listOrders  *usecase.ListOrdersUseCase
	getOrder    *usecase.GetOrderUseCase
	createOrder *usecase.CreateOrderUseCase
	log         *logrus.Logger
}

func NewHandler(
	listOrders *usecase.ListOrdersUseCase,
	getOrder *usecase.GetOrderUseCase,
	createOrder *usecase.CreateOrderUseCase,
	log *logrus.Logger,
) *Handler {
	return &Handler{
		listOrders:  listOrders,
		getOrder:    getOrder,
		createOrder: createOrder,
		log:         log,
	}
}
