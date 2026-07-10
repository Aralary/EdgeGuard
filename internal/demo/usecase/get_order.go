package usecase

import (
	"context"

	"github.com/aralary/edgeguard/internal/demo/domain"
)

func (u *Usecase) GetOrder(ctx context.Context, id string) (domain.Order, error) {
	return u.orderRepository.GetOrderByID(ctx, id)
}
