package memory

import (
	"context"
	"errors"
	"testing"

	"github.com/aralary/edgeguard/internal/demo/domain"
)

func TestOrderRepositoryListOrders(t *testing.T) {
	repo := NewOrderRepository()

	orders, err := repo.ListOrders(context.Background())
	if err != nil {
		t.Fatalf("ListOrders() unexpected error = %v", err)
	}

	if len(orders) != 2 {
		t.Fatalf("len(orders) = %d, want 2", len(orders))
	}

	assertOrderExists(t, orders, "ord_1")
	assertOrderExists(t, orders, "ord_2")
}

func TestOrderRepositoryGetOrderByID(t *testing.T) {
	repo := NewOrderRepository()

	order, err := repo.GetOrderByID(context.Background(), "ord_1")
	if err != nil {
		t.Fatalf("GetOrderByID() unexpected error = %v", err)
	}

	if order.ID != "ord_1" {
		t.Fatalf("order.ID = %q, want %q", order.ID, "ord_1")
	}
}

func TestOrderRepositoryGetOrderByIDReturnsNotFound(t *testing.T) {
	repo := NewOrderRepository()

	_, err := repo.GetOrderByID(context.Background(), "missing")
	if !errors.Is(err, ErrOrderNotFound) {
		t.Fatalf("GetOrderByID() error = %v, want %v", err, ErrOrderNotFound)
	}
}

func TestOrderRepositoryCreateOrder(t *testing.T) {
	repo := NewOrderRepository()
	created := domain.Order{
		ID:     "ord_3",
		Status: "created",
		Amount: 5000,
	}

	order, err := repo.CreateOrder(context.Background(), created)
	if err != nil {
		t.Fatalf("CreateOrder() unexpected error = %v", err)
	}

	if order != created {
		t.Fatalf("CreateOrder() = %+v, want %+v", order, created)
	}

	stored, err := repo.GetOrderByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetOrderByID() unexpected error = %v", err)
	}

	if stored != created {
		t.Fatalf("stored order = %+v, want %+v", stored, created)
	}
}

func assertOrderExists(t *testing.T, orders []domain.Order, id string) {
	t.Helper()

	for _, order := range orders {
		if order.ID == id {
			return
		}
	}

	t.Fatalf("order %q not found in %+v", id, orders)
}
