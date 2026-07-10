package usecase

import (
	"context"

	"github.com/aralary/edgeguard/internal/gateway/domain"
)

func (u *Usecase) ResolveRoute(ctx context.Context, path string) (domain.Route, error) {
	routes, err := u.routeRepository.ListRoutes(ctx)
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
