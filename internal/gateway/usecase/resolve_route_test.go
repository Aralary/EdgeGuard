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

func (r fakeRouteRepository) ReplaceRoutes([]domain.Route) {}

func TestUsecaseResolveRoute(t *testing.T) {
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
			uc := New(fakeRouteRepository{routes: tt.routes}, nil, nil, nil)

			got, err := uc.ResolveRoute(context.Background(), tt.path)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("ResolveRoute() error = %v, want %v", err, tt.wantErr)
				}

				return
			}

			if err != nil {
				t.Fatalf("ResolveRoute() unexpected error = %v", err)
			}

			if got.Name != tt.wantRoute {
				t.Fatalf("ResolveRoute() route = %q, want %q", got.Name, tt.wantRoute)
			}
		})
	}
}

func TestUsecaseResolveRouteReturnsRepositoryError(t *testing.T) {
	repoErr := errors.New("repository failed")
	uc := New(fakeRouteRepository{err: repoErr}, nil, nil, nil)

	_, err := uc.ResolveRoute(context.Background(), "/api/v1/orders")
	if !errors.Is(err, repoErr) {
		t.Fatalf("ResolveRoute() error = %v, want %v", err, repoErr)
	}
}
