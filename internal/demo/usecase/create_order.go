package usecase

import (
	"context"

	"github.com/aralary/edgeguard/internal/demo/domain"
)

func (u *Usecase) CreateOrder(ctx context.Context, order domain.Order) (domain.Order, error) {
	return u.orderRepository.CreateOrder(ctx, order)
}
