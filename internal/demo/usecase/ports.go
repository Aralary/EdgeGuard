package usecase

import (
	"context"

	"github.com/aralary/edgeguard/internal/demo/domain"
)

type OrderRepository interface {
	ListOrders(ctx context.Context) ([]domain.Order, error)
	GetOrderByID(ctx context.Context, id string) (domain.Order, error)
	CreateOrder(ctx context.Context, order domain.Order) (domain.Order, error)
}
