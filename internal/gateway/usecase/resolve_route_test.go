package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/aralary/edgeguard/internal/gateway/domain"
)

type fakeRouteRepository struct {
	routes []domain.Route
	err    error
}

func (r fakeRouteRepository) ListRoutes(ctx context.Context) ([]domain.Route, error) {
	if r.err != nil {
		return nil, r.err
	}

	return r.routes, nil
}

func TestResolveRouteUseCaseExecute(t *testing.T) {
	tests := []struct {
		name      string
		routes    []domain.Route
		path      string
		wantRoute string
		wantErr   error
	}{
		{
			name: "returns matching route",
			routes: []domain.Route{
				{Name: "demo-api-v1", PathPrefix: "/api/v1"},
			},
			path:      "/api/v1/orders",
			wantRoute: "demo-api-v1",
		},
		{
			name: "selects longest matching prefix",
			routes: []domain.Route{
				{Name: "api", PathPrefix: "/api"},
				{Name: "api-v1", PathPrefix: "/api/v1"},
				{Name: "orders", PathPrefix: "/api/v1/orders"},
			},
			path:      "/api/v1/orders/ord_1",
			wantRoute: "orders",
		},
		{
			name: "does not match similar prefix without path boundary",
			routes: []domain.Route{
				{Name: "api-v1", PathPrefix: "/api/v1"},
			},
			path:    "/api/v10/orders",
			wantErr: domain.ErrRouteNotFound,
		},
		{
			name:    "returns route not found for empty route list",
			routes:  nil,
			path:    "/api/v1/orders",
			wantErr: domain.ErrRouteNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := NewResolveRouteUseCase(fakeRouteRepository{routes: tt.routes})

			got, err := uc.Execute(context.Background(), tt.path)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Execute() error = %v, want %v", err, tt.wantErr)
				}

				return
			}

			if err != nil {
				t.Fatalf("Execute() unexpected error = %v", err)
			}

			if got.Name != tt.wantRoute {
				t.Fatalf("Execute() route = %q, want %q", got.Name, tt.wantRoute)
			}
		})
	}
}

func TestResolveRouteUseCaseExecuteReturnsRepositoryError(t *testing.T) {
	repoErr := errors.New("repository failed")
	uc := NewResolveRouteUseCase(fakeRouteRepository{err: repoErr})

	_, err := uc.Execute(context.Background(), "/api/v1/orders")
	if !errors.Is(err, repoErr) {
		t.Fatalf("Execute() error = %v, want %v", err, repoErr)
	}
}
