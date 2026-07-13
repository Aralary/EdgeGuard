package usecase

import (
	"context"
	"strings"

	"github.com/aralary/edgeguard/internal/jobs/domain"
)

func (u *Usecase) GetJob(ctx context.Context, jobID string) (domain.Job, error) {
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return domain.Job{}, domain.ErrJobNotFound
	}

	return u.repository.GetJob(ctx, jobID)
}
