package usecase

import (
	"context"
	"errors"
	"fmt"
)

var ErrRouteSourceNotConfigured = errors.New("route source is not configured")

// RefreshRoutes loads a complete route snapshot from the configured source and
// atomically replaces the current repository snapshot. If loading fails, the
// current repository state is left unchanged.
func (u *Usecase) RefreshRoutes(ctx context.Context) (int, error) {
	if u.routeSource == nil {
		return 0, ErrRouteSourceNotConfigured
	}

	routes, err := u.routeSource.ListRoutes(ctx)
	if err != nil {
		return 0, fmt.Errorf("load route snapshot: %w", err)
	}

	u.routeRepository.ReplaceRoutes(routes)

	return len(routes), nil
}
