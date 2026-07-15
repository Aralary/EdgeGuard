package memory

import (
	"context"
	"errors"
	"sync"

	"github.com/aralary/edgeguard/internal/demo/domain"
)

var ErrOrderNotFound = errors.New("order not found")

type OrderRepository struct {
	mu     sync.RWMutex
	orders map[string]domain.Order
}

func NewOrderRepository() *OrderRepository {
	return &OrderRepository{
		orders: map[string]domain.Order{
			"ord_1": {
				ID:     "ord_1",
				Status: "created",
				Amount: 1000,
			},
			"ord_2": {
				ID:     "ord_2",
				Status: "paid",
				Amount: 2500,
			},
		},
	}
}

func (r *OrderRepository) ListOrders(ctx context.Context) ([]domain.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]domain.Order, 0, len(r.orders))
	for _, order := range r.orders {
		result = append(result, order)
	}

	return result, nil
}

func (r *OrderRepository) GetOrderByID(ctx context.Context, id string) (domain.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	order, ok := r.orders[id]
	if !ok {
		return domain.Order{}, ErrOrderNotFound
	}

	return order, nil
}

func (r *OrderRepository) CreateOrder(ctx context.Context, order domain.Order) (domain.Order, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.orders[order.ID] = order

	return order, nil
}
