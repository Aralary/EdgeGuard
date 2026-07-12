package memory

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/aralary/edgeguard/internal/gateway/domain"
)

func TestNewRouteRepositoryCopiesInitialRoutes(t *testing.T) {
	initial := []domain.Route{
		{Name: "demo-api-v1", PathPrefix: "/api/v1"},
	}

	repository := NewRouteRepository(initial)
	initial[0].Name = "changed"

	routes, err := repository.ListRoutes(context.Background())
	if err != nil {
		t.Fatalf("ListRoutes() unexpected error = %v", err)
	}

	if routes[0].Name != "demo-api-v1" {
		t.Fatalf("stored route name = %q, want %q", routes[0].Name, "demo-api-v1")
	}
}

func TestRouteRepositoryListRoutesReturnsCopy(t *testing.T) {
	repository := NewRouteRepository([]domain.Route{
		{Name: "demo-api-v1", PathPrefix: "/api/v1"},
	})

	first, err := repository.ListRoutes(context.Background())
	if err != nil {
		t.Fatalf("ListRoutes() unexpected error = %v", err)
	}

	first[0].Name = "changed"

	second, err := repository.ListRoutes(context.Background())
	if err != nil {
		t.Fatalf("ListRoutes() unexpected error = %v", err)
	}

	if second[0].Name != "demo-api-v1" {
		t.Fatalf("repository returned mutable internal slice, got route name %q", second[0].Name)
	}
}

func TestRouteRepositoryReplaceRoutes(t *testing.T) {
	repository := NewRouteRepository([]domain.Route{
		{Name: "old", PathPrefix: "/old"},
	})

	next := []domain.Route{
		{Name: "new", PathPrefix: "/new"},
	}
	repository.ReplaceRoutes(next)
	next[0].Name = "changed"

	routes, err := repository.ListRoutes(context.Background())
	if err != nil {
		t.Fatalf("ListRoutes() unexpected error = %v", err)
	}

	if len(routes) != 1 {
		t.Fatalf("len(routes) = %d, want 1", len(routes))
	}

	if routes[0].Name != "new" {
		t.Fatalf("route name = %q, want %q", routes[0].Name, "new")
	}
}

func TestRouteRepositoryReplaceRoutesCanClearSnapshot(t *testing.T) {
	repository := NewRouteRepository([]domain.Route{
		{Name: "demo-api-v1", PathPrefix: "/api/v1"},
	})

	repository.ReplaceRoutes(nil)

	routes, err := repository.ListRoutes(context.Background())
	if err != nil {
		t.Fatalf("ListRoutes() unexpected error = %v", err)
	}

	if len(routes) != 0 {
		t.Fatalf("len(routes) = %d, want 0", len(routes))
	}
}

func TestRouteRepositoryConcurrentAccess(t *testing.T) {
	repository := NewRouteRepository(nil)

	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		for i := 0; i < iterations; i++ {
			repository.ReplaceRoutes([]domain.Route{
				{Name: fmt.Sprintf("route-%d", i), PathPrefix: "/api"},
			})
		}
	}()

	go func() {
		defer wg.Done()

		for i := 0; i < iterations; i++ {
			if _, err := repository.ListRoutes(context.Background()); err != nil {
				t.Errorf("ListRoutes() unexpected error = %v", err)
				return
			}
		}
	}()

	wg.Wait()
}
