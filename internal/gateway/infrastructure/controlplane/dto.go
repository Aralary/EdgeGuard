package controlplane

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/aralary/edgeguard/internal/gateway/domain"
)

type routeResponse struct {
	Name        string `json:"name"`
	PathPrefix  string `json:"path_prefix"`
	UpstreamURL string `json:"upstream_url"`
	StripPrefix bool   `json:"strip_prefix"`
	TimeoutMS   int    `json:"timeout_ms"`
}

func responseToDomain(response []routeResponse) ([]domain.Route, error) {
	routes := make([]domain.Route, 0, len(response))
	pathPrefixes := make(map[string]struct{}, len(response))

	for i, item := range response {
		route, err := item.toDomain()
		if err != nil {
			return nil, fmt.Errorf("invalid route at index %d: %w", i, err)
		}

		if _, exists := pathPrefixes[route.PathPrefix]; exists {
			return nil, fmt.Errorf("invalid route at index %d: duplicate path prefix %q", i, route.PathPrefix)
		}

		pathPrefixes[route.PathPrefix] = struct{}{}
		routes = append(routes, route)
	}

	return routes, nil
}

func (r routeResponse) toDomain() (domain.Route, error) {
	name := strings.TrimSpace(r.Name)
	if name == "" {
		return domain.Route{}, fmt.Errorf("name is required")
	}

	pathPrefix := normalizePathPrefix(r.PathPrefix)
	if pathPrefix == "" || !strings.HasPrefix(pathPrefix, "/") {
		return domain.Route{}, fmt.Errorf("path prefix must start with /")
	}

	upstreamURL := strings.TrimSpace(r.UpstreamURL)
	parsedUpstream, err := url.Parse(upstreamURL)
	if err != nil {
		return domain.Route{}, fmt.Errorf("parse upstream url: %w", err)
	}

	if (parsedUpstream.Scheme != "http" && parsedUpstream.Scheme != "https") || parsedUpstream.Host == "" {
		return domain.Route{}, fmt.Errorf("upstream url must be an absolute http or https url")
	}

	if r.TimeoutMS <= 0 {
		return domain.Route{}, fmt.Errorf("timeout_ms must be greater than zero")
	}

	return domain.Route{
		Name:        name,
		PathPrefix:  pathPrefix,
		UpstreamURL: upstreamURL,
		StripPrefix: r.StripPrefix,
		Timeout:     time.Duration(r.TimeoutMS) * time.Millisecond,
	}, nil
}

func normalizePathPrefix(pathPrefix string) string {
	pathPrefix = strings.TrimSpace(pathPrefix)
	if pathPrefix == "/" {
		return pathPrefix
	}

	return strings.TrimRight(pathPrefix, "/")
}
