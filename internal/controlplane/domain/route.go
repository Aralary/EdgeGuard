package domain

import (
	"strings"
	"time"
)

const DefaultRouteTimeoutMS = 3000

type Route struct {
	ID           string
	ServiceID    string
	Name         string
	PathPrefix   string
	StripPrefix  bool
	TimeoutMS    int
	Enabled      bool
	AuthRequired bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func NewRoute(
	serviceID string,
	name string,
	pathPrefix string,
	stripPrefix bool,
	timeoutMS int,
	enabled bool,
	authRequired bool,
) (Route, error) {
	serviceID = strings.TrimSpace(serviceID)
	name = strings.TrimSpace(name)
	pathPrefix = strings.TrimSpace(pathPrefix)

	if serviceID == "" {
		return Route{}, ErrInvalidServiceID
	}

	if name == "" {
		return Route{}, ErrInvalidName
	}

	if pathPrefix == "" || !strings.HasPrefix(pathPrefix, "/") {
		return Route{}, ErrInvalidPathPrefix
	}

	if timeoutMS <= 0 {
		timeoutMS = DefaultRouteTimeoutMS
	}

	return Route{
		ServiceID:    serviceID,
		Name:         name,
		PathPrefix:   pathPrefix,
		StripPrefix:  stripPrefix,
		TimeoutMS:    timeoutMS,
		Enabled:      enabled,
		AuthRequired: authRequired,
	}, nil
}
