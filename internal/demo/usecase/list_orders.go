package usecase

import (
	"context"

	"github.com/aralary/edgeguard/internal/demo/domain"
)

func (u *Usecase) ListOrders(ctx context.Context) ([]domain.Order, error) {
	return u.orderRepository.ListOrders(ctx)
}
