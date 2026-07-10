package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/aralary/edgeguard/internal/demo/domain"
)

type fakeOrderRepository struct {
	orders []domain.Order
	order  domain.Order
	err    error
}

func (r fakeOrderRepository) ListOrders(ctx context.Context) ([]domain.Order, error) {
	if r.err != nil {
		return nil, r.err
	}

	return r.orders, nil
}

func (r fakeOrderRepository) GetOrderByID(ctx context.Context, id string) (domain.Order, error) {
	if r.err != nil {
		return domain.Order{}, r.err
	}

	return r.order, nil
}

func (r fakeOrderRepository) CreateOrder(ctx context.Context, order domain.Order) (domain.Order, error) {
	if r.err != nil {
		return domain.Order{}, r.err
	}

	return order, nil
}

func TestUsecaseListOrders(t *testing.T) {
	orders := []domain.Order{{ID: "ord_1", Status: "created", Amount: 1000}}
	uc := New(fakeOrderRepository{orders: orders})

	got, err := uc.ListOrders(context.Background())
	if err != nil {
		t.Fatalf("unexpected error = %v", err)
	}

	if len(got) != 1 || got[0] != orders[0] {
		t.Fatalf("result = %+v, want %+v", got, orders)
	}
}

func TestUsecaseGetOrder(t *testing.T) {
	order := domain.Order{ID: "ord_1", Status: "created", Amount: 1000}
	uc := New(fakeOrderRepository{order: order})

	got, err := uc.GetOrder(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("unexpected error = %v", err)
	}

	if got != order {
		t.Fatalf("result = %+v, want %+v", got, order)
	}
}

func TestUsecaseCreateOrder(t *testing.T) {
	order := domain.Order{ID: "ord_1", Status: "created", Amount: 1000}
	uc := New(fakeOrderRepository{})

	got, err := uc.CreateOrder(context.Background(), order)
	if err != nil {
		t.Fatalf("unexpected error = %v", err)
	}

	if got != order {
		t.Fatalf("result = %+v, want %+v", got, order)
	}
}

func TestUsecaseReturnsRepositoryError(t *testing.T) {
	repoErr := errors.New("repository failed")

	if _, err := New(fakeOrderRepository{err: repoErr}).ListOrders(context.Background()); !errors.Is(err, repoErr) {
		t.Fatalf("ListOrders error = %v, want %v", err, repoErr)
	}

	if _, err := New(fakeOrderRepository{err: repoErr}).GetOrder(context.Background(), "ord_1"); !errors.Is(err, repoErr) {
		t.Fatalf("GetOrder error = %v, want %v", err, repoErr)
	}

	if _, err := New(fakeOrderRepository{err: repoErr}).CreateOrder(context.Background(), domain.Order{}); !errors.Is(err, repoErr) {
		t.Fatalf("CreateOrder error = %v, want %v", err, repoErr)
	}
}
