package memory

import (
	"context"
	"sync"

	"github.com/aralary/edgeguard/internal/gateway/domain"
)

// RouteRepository stores the current gateway route snapshot in memory.
// Replacing the snapshot and reading it are safe for concurrent use.
type RouteRepository struct {
	mu     sync.RWMutex
	routes []domain.Route
}

func NewRouteRepository(routes []domain.Route) *RouteRepository {
	return &RouteRepository{
		routes: cloneRoutes(routes),
	}
}

func (r *RouteRepository) ListRoutes(context.Context) ([]domain.Route, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return cloneRoutes(r.routes), nil
}

// ReplaceRoutes atomically replaces the complete route snapshot.
func (r *RouteRepository) ReplaceRoutes(routes []domain.Route) {
	next := cloneRoutes(routes)

	r.mu.Lock()
	r.routes = next
	r.mu.Unlock()
}

func cloneRoutes(routes []domain.Route) []domain.Route {
	if routes == nil {
		return nil
	}

	result := make([]domain.Route, len(routes))
	copy(result, routes)

	return result
}
