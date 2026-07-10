package httpdelivery

import (
	"context"

	"github.com/aralary/edgeguard/internal/controlplane/domain"
	"github.com/aralary/edgeguard/internal/controlplane/usecase"
)

type ControlPlaneUsecase interface {
	CreateProject(ctx context.Context, input usecase.CreateProjectInput) (domain.Project, error)
	ListProjects(ctx context.Context) ([]domain.Project, error)
	CreateService(ctx context.Context, input usecase.CreateServiceInput) (domain.Service, error)
	ListServices(ctx context.Context, projectID string) ([]domain.Service, error)
	CreateRoute(ctx context.Context, input usecase.CreateRouteInput) (domain.Route, error)
	ListRoutes(ctx context.Context, serviceID string) ([]domain.Route, error)
}
