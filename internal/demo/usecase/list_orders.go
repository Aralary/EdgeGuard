package usecase

import (
	"context"

	"github.com/aralary/edgeguard/internal/demo/domain"
)

type ListOrdersUseCase struct {
	orders OrderRepository
}

func NewListOrdersUseCase(orders OrderRepository) *ListOrdersUseCase {
	return &ListOrdersUseCase{orders: orders}
}

func (uc *ListOrdersUseCase) Execute(ctx context.Context) ([]domain.Order, error) {
	return uc.orders.ListOrders(ctx)
}