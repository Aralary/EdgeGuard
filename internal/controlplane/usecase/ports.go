package usecase

import (
	"context"

	"github.com/aralary/edgeguard/internal/controlplane/domain"
)

type ProjectRepository interface {
	CreateProject(ctx context.Context, project domain.Project) (domain.Project, error)
	ListProjects(ctx context.Context) ([]domain.Project, error)
}

type ServiceRepository interface {
	CreateService(ctx context.Context, service domain.Service) (domain.Service, error)
	ListServicesByProjectID(ctx context.Context, projectID string) ([]domain.Service, error)
}

type RouteRepository interface {
	CreateRoute(ctx context.Context, route domain.Route) (domain.Route, error)
	ListRoutesByServiceID(ctx context.Context, serviceID string) ([]domain.Route, error)
}
