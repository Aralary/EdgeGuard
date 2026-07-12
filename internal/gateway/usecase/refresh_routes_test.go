package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/aralary/edgeguard/internal/gateway/domain"
)

type refreshRouteRepository struct {
	routes       []domain.Route
	replaceCalls int
}

func (r *refreshRouteRepository) ListRoutes(context.Context) ([]domain.Route, error) {
	return r.routes, nil
}

func (r *refreshRouteRepository) ReplaceRoutes(routes []domain.Route) {
	r.replaceCalls++
	r.routes = append([]domain.Route(nil), routes...)
}

type fakeRouteSource struct {
	routes []domain.Route
	err    error
}

func (s fakeRouteSource) ListRoutes(context.Context) ([]domain.Route, error) {
	if s.err != nil {
		return nil, s.err
	}

	return s.routes, nil
}

func TestUsecaseRefreshRoutesReplacesSnapshot(t *testing.T) {
	repository := &refreshRouteRepository{
		routes: []domain.Route{{Name: "old", PathPrefix: "/old"}},
	}
	source := fakeRouteSource{
		routes: []domain.Route{{Name: "new", PathPrefix: "/new"}},
	}

	uc := New(repository, source)

	count, err := uc.RefreshRoutes(context.Background())
	if err != nil {
		t.Fatalf("RefreshRoutes() unexpected error = %v", err)
	}

	if count != 1 {
		t.Fatalf("RefreshRoutes() count = %d, want 1", count)
	}

	if repository.replaceCalls != 1 {
		t.Fatalf("ReplaceRoutes() calls = %d, want 1", repository.replaceCalls)
	}

	if len(repository.routes) != 1 || repository.routes[0].Name != "new" {
		t.Fatalf("repository routes = %#v, want new snapshot", repository.routes)
	}
}

func TestUsecaseRefreshRoutesCanClearSnapshot(t *testing.T) {
	repository := &refreshRouteRepository{
		routes: []domain.Route{{Name: "old", PathPrefix: "/old"}},
	}

	uc := New(repository, fakeRouteSource{routes: []domain.Route{}})

	count, err := uc.RefreshRoutes(context.Background())
	if err != nil {
		t.Fatalf("RefreshRoutes() unexpected error = %v", err)
	}

	if count != 0 {
		t.Fatalf("RefreshRoutes() count = %d, want 0", count)
	}

	if len(repository.routes) != 0 {
		t.Fatalf("len(repository.routes) = %d, want 0", len(repository.routes))
	}
}

func TestUsecaseRefreshRoutesPreservesSnapshotOnSourceError(t *testing.T) {
	repository := &refreshRouteRepository{
		routes: []domain.Route{{Name: "working", PathPrefix: "/api"}},
	}
	sourceErr := errors.New("control plane unavailable")

	uc := New(repository, fakeRouteSource{err: sourceErr})

	_, err := uc.RefreshRoutes(context.Background())
	if !errors.Is(err, sourceErr) {
		t.Fatalf("RefreshRoutes() error = %v, want %v", err, sourceErr)
	}

	if repository.replaceCalls != 0 {
		t.Fatalf("ReplaceRoutes() calls = %d, want 0", repository.replaceCalls)
	}

	if len(repository.routes) != 1 || repository.routes[0].Name != "working" {
		t.Fatalf("repository routes = %#v, want previous snapshot", repository.routes)
	}
}

func TestUsecaseRefreshRoutesWithoutSource(t *testing.T) {
	repository := &refreshRouteRepository{}
	uc := New(repository, nil)

	_, err := uc.RefreshRoutes(context.Background())
	if !errors.Is(err, ErrRouteSourceNotConfigured) {
		t.Fatalf("RefreshRoutes() error = %v, want %v", err, ErrRouteSourceNotConfigured)
	}
}
