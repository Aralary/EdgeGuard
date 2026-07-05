package usecase

import (
	"context"

	"github.com/aralary/edgeguard/internal/demo/domain"
)

type CreateOrderUseCase struct {
	orders OrderRepository
}

func NewCreateOrderUseCase(orders OrderRepository) *CreateOrderUseCase {
	return &CreateOrderUseCase{orders: orders}
}

func (uc *CreateOrderUseCase) Execute(ctx context.Context, order domain.Order) (domain.Order, error) {
	return uc.orders.CreateOrder(ctx, order)
}