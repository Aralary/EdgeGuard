package usecase

import (
	"context"
	"strings"

	"github.com/aralary/edgeguard/internal/controlplane/domain"
)

type CreateRouteInput struct {
	ServiceID   string
	Name        string
	PathPrefix  string
	StripPrefix bool
	TimeoutMS   int
	Enabled     bool
}

func (u *Usecase) CreateRoute(ctx context.Context, input CreateRouteInput) (domain.Route, error) {
	route, err := domain.NewRoute(input.ServiceID, input.Name, input.PathPrefix, input.StripPrefix, input.TimeoutMS, input.Enabled)
	if err != nil {
		return domain.Route{}, err
	}

	return u.routeRepository.CreateRoute(ctx, route)
}

func (u *Usecase) ListRoutes(ctx context.Context, serviceID string) ([]domain.Route, error) {
	serviceID = strings.TrimSpace(serviceID)
	if serviceID == "" {
		return nil, domain.ErrInvalidServiceID
	}

	return u.routeRepository.ListRoutesByServiceID(ctx, serviceID)
}

func (u *Usecase) ListGatewayRoutes(ctx context.Context) ([]domain.GatewayRoute, error) {
	return u.routeRepository.ListGatewayRoutes(ctx)
}
