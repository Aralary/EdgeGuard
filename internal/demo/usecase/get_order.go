package usecase

import (
	"context"

	"github.com/aralary/edgeguard/internal/demo/domain"
)

type GetOrderUseCase struct {
	orders OrderRepository
}

func NewGetOrderUseCase(orders OrderRepository) *GetOrderUseCase {
	return &GetOrderUseCase{orders: orders}
}

func (uc *GetOrderUseCase) Execute(ctx context.Context, id string) (domain.Order, error) {
	return uc.orders.GetOrderByID(ctx, id)
}