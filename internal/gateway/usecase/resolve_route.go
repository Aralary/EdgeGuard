package usecase

import (
	"context"

	"github.com/aralary/edgeguard/internal/gateway/domain"
)

type ResolveRouteUseCase struct {
	routes RouteRepository
}

func NewResolveRouteUseCase(routes RouteRepository) *ResolveRouteUseCase {
	return &ResolveRouteUseCase{
		routes: routes,
	}
}

func (uc *ResolveRouteUseCase) Execute(ctx context.Context, path string) (domain.Route, error) {
	routes, err := uc.routes.ListRoutes(ctx)
	if err != nil {
		return domain.Route{}, err
	}

	var matched domain.Route
	matchedLen := -1

	for _, route := range routes {
		if route.Matches(path) && len(route.PathPrefix) > matchedLen {
			matched = route
			matchedLen = len(route.PathPrefix)
		}
	}

	if matchedLen == -1 {
		return domain.Route{}, domain.ErrRouteNotFound
	}

	return matched, nil
}