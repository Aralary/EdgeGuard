package httpdelivery

import (
	"time"

	"github.com/aralary/edgeguard/internal/controlplane/domain"
)

type createProjectRequest struct {
	Name string `json:"name"`
}

type createServiceRequest struct {
	Name        string `json:"name"`
	UpstreamURL string `json:"upstream_url"`
}

type createRouteRequest struct {
	Name        string `json:"name"`
	PathPrefix  string `json:"path_prefix"`
	StripPrefix *bool  `json:"strip_prefix"`
	TimeoutMS   int    `json:"timeout_ms"`
	Enabled     *bool  `json:"enabled"`
}

type projectResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type serviceResponse struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	Name        string    `json:"name"`
	UpstreamURL string    `json:"upstream_url"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type routeResponse struct {
	ID          string    `json:"id"`
	ServiceID   string    `json:"service_id"`
	Name        string    `json:"name"`
	PathPrefix  string    `json:"path_prefix"`
	StripPrefix bool      `json:"strip_prefix"`
	TimeoutMS   int       `json:"timeout_ms"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func newProjectResponse(project domain.Project) projectResponse {
	return projectResponse{
		ID:        project.ID,
		Name:      project.Name,
		CreatedAt: project.CreatedAt,
		UpdatedAt: project.UpdatedAt,
	}
}

func newProjectListResponse(projects []domain.Project) []projectResponse {
	response := make([]projectResponse, 0, len(projects))
	for _, project := range projects {
		response = append(response, newProjectResponse(project))
	}

	return response
}

func newServiceResponse(service domain.Service) serviceResponse {
	return serviceResponse{
		ID:          service.ID,
		ProjectID:   service.ProjectID,
		Name:        service.Name,
		UpstreamURL: service.UpstreamURL,
		CreatedAt:   service.CreatedAt,
		UpdatedAt:   service.UpdatedAt,
	}
}

func newServiceListResponse(services []domain.Service) []serviceResponse {
	response := make([]serviceResponse, 0, len(services))
	for _, service := range services {
		response = append(response, newServiceResponse(service))
	}

	return response
}

func newRouteResponse(route domain.Route) routeResponse {
	return routeResponse{
		ID:          route.ID,
		ServiceID:   route.ServiceID,
		Name:        route.Name,
		PathPrefix:  route.PathPrefix,
		StripPrefix: route.StripPrefix,
		TimeoutMS:   route.TimeoutMS,
		Enabled:     route.Enabled,
		CreatedAt:   route.CreatedAt,
		UpdatedAt:   route.UpdatedAt,
	}
}

func newRouteListResponse(routes []domain.Route) []routeResponse {
	response := make([]routeResponse, 0, len(routes))
	for _, route := range routes {
		response = append(response, newRouteResponse(route))
	}

	return response
}

type gatewayRouteResponse struct {
	Name        string `json:"name"`
	PathPrefix  string `json:"path_prefix"`
	UpstreamURL string `json:"upstream_url"`
	StripPrefix bool   `json:"strip_prefix"`
	TimeoutMS   int    `json:"timeout_ms"`
}

func newGatewayRouteResponse(route domain.GatewayRoute) gatewayRouteResponse {
	return gatewayRouteResponse{
		Name:        route.Name,
		PathPrefix:  route.PathPrefix,
		UpstreamURL: route.UpstreamURL,
		StripPrefix: route.StripPrefix,
		TimeoutMS:   route.TimeoutMS,
	}
}

func newGatewayRouteListResponse(routes []domain.GatewayRoute) []gatewayRouteResponse {
	response := make([]gatewayRouteResponse, 0, len(routes))
	for _, route := range routes {
		response = append(response, newGatewayRouteResponse(route))
	}

	return response
}
