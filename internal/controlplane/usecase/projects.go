package usecase

import (
	"context"

	"github.com/aralary/edgeguard/internal/controlplane/domain"
)

type CreateProjectInput struct {
	Name string
}

func (u *Usecase) CreateProject(ctx context.Context, input CreateProjectInput) (domain.Project, error) {
	project, err := domain.NewProject(input.Name)
	if err != nil {
		return domain.Project{}, err
	}

	return u.projectRepository.CreateProject(ctx, project)
}

func (u *Usecase) ListProjects(ctx context.Context) ([]domain.Project, error) {
	return u.projectRepository.ListProjects(ctx)
}
