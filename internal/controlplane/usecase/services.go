package usecase

import (
	"context"
	"strings"

	"github.com/aralary/edgeguard/internal/controlplane/domain"
)

type CreateServiceInput struct {
	ProjectID   string
	Name        string
	UpstreamURL string
}

func (u *Usecase) CreateService(ctx context.Context, input CreateServiceInput) (domain.Service, error) {
	service, err := domain.NewService(input.ProjectID, input.Name, input.UpstreamURL)
	if err != nil {
		return domain.Service{}, err
	}

	return u.serviceRepository.CreateService(ctx, service)
}

func (u *Usecase) ListServices(ctx context.Context, projectID string) ([]domain.Service, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, domain.ErrInvalidProjectID
	}

	return u.serviceRepository.ListServicesByProjectID(ctx, projectID)
}
