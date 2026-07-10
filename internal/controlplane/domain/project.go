package domain

import (
	"strings"
	"time"
)

type Project struct {
	ID        string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewProject(name string) (Project, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Project{}, ErrInvalidName
	}

	return Project{
		Name: name,
	}, nil
}
