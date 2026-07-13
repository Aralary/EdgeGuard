package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/aralary/edgeguard/internal/jobs/domain"
)

func TestGetJob(t *testing.T) {
	repository := &jobRepositoryStub{job: domain.Job{ID: "job-1", Status: domain.StatusSucceeded}}
	uc := New(Dependencies{Repository: repository})

	job, err := uc.GetJob(context.Background(), " job-1 ")
	if err != nil {
		t.Fatalf("GetJob() error = %v", err)
	}
	if job.ID != "job-1" || job.Status != domain.StatusSucceeded {
		t.Fatalf("job = %#v", job)
	}
}

func TestGetJobEmptyID(t *testing.T) {
	uc := New(Dependencies{Repository: &jobRepositoryStub{}})

	_, err := uc.GetJob(context.Background(), " ")
	if !errors.Is(err, domain.ErrJobNotFound) {
		t.Fatalf("GetJob() error = %v, want %v", err, domain.ErrJobNotFound)
	}
}
