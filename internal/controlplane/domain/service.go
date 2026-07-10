package domain

import (
	"net/url"
	"strings"
	"time"
)

type Service struct {
	ID          string
	ProjectID   string
	Name        string
	UpstreamURL string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func NewService(projectID string, name string, upstreamURL string) (Service, error) {
	projectID = strings.TrimSpace(projectID)
	name = strings.TrimSpace(name)
	upstreamURL = strings.TrimSpace(upstreamURL)

	if projectID == "" {
		return Service{}, ErrInvalidProjectID
	}

	if name == "" {
		return Service{}, ErrInvalidName
	}

	parsed, err := url.Parse(upstreamURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return Service{}, ErrInvalidUpstreamURL
	}

	return Service{
		ProjectID:   projectID,
		Name:        name,
		UpstreamURL: upstreamURL,
	}, nil
}
