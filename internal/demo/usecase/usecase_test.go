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

func TestListOrdersUseCaseExecute(t *testing.T) {
	orders := []domain.Order{{ID: "ord_1", Status: "created", Amount: 1000}}
	uc := NewListOrdersUseCase(fakeOrderRepository{orders: orders})

	got, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute() unexpected error = %v", err)
	}

	if len(got) != 1 || got[0] != orders[0] {
		t.Fatalf("Execute() = %+v, want %+v", got, orders)
	}
}

func TestGetOrderUseCaseExecute(t *testing.T) {
	order := domain.Order{ID: "ord_1", Status: "created", Amount: 1000}
	uc := NewGetOrderUseCase(fakeOrderRepository{order: order})

	got, err := uc.Execute(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("Execute() unexpected error = %v", err)
	}

	if got != order {
		t.Fatalf("Execute() = %+v, want %+v", got, order)
	}
}

func TestCreateOrderUseCaseExecute(t *testing.T) {
	order := domain.Order{ID: "ord_1", Status: "created", Amount: 1000}
	uc := NewCreateOrderUseCase(fakeOrderRepository{})

	got, err := uc.Execute(context.Background(), order)
	if err != nil {
		t.Fatalf("Execute() unexpected error = %v", err)
	}

	if got != order {
		t.Fatalf("Execute() = %+v, want %+v", got, order)
	}
}

func TestUseCasesReturnRepositoryError(t *testing.T) {
	repoErr := errors.New("repository failed")

	if _, err := NewListOrdersUseCase(fakeOrderRepository{err: repoErr}).Execute(context.Background()); !errors.Is(err, repoErr) {
		t.Fatalf("ListOrdersUseCase error = %v, want %v", err, repoErr)
	}

	if _, err := NewGetOrderUseCase(fakeOrderRepository{err: repoErr}).Execute(context.Background(), "ord_1"); !errors.Is(err, repoErr) {
		t.Fatalf("GetOrderUseCase error = %v, want %v", err, repoErr)
	}

	if _, err := NewCreateOrderUseCase(fakeOrderRepository{err: repoErr}).Execute(context.Background(), domain.Order{}); !errors.Is(err, repoErr) {
		t.Fatalf("CreateOrderUseCase error = %v, want %v", err, repoErr)
	}
}
