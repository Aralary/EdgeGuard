package httpdelivery

import (
	"encoding/json"
	"net/http"

	"github.com/aralary/edgeguard/internal/controlplane/usecase"
	"github.com/labstack/echo/v5"
)

func (h *Handler) createRoute(c *echo.Context) error {
	var request createRouteRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&request); err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid request body"})
	}

	stripPrefix := true
	if request.StripPrefix != nil {
		stripPrefix = *request.StripPrefix
	}

	enabled := true
	if request.Enabled != nil {
		enabled = *request.Enabled
	}

	route, err := h.usecase.CreateRoute(c.Request().Context(), usecase.CreateRouteInput{
		ServiceID:   c.Param("service_id"),
		Name:        request.Name,
		PathPrefix:  request.PathPrefix,
		StripPrefix: stripPrefix,
		TimeoutMS:   request.TimeoutMS,
		Enabled:     enabled,
	})
	if err != nil {
		return h.handleError(c, "failed to create route", err)
	}

	return c.JSON(http.StatusCreated, newRouteResponse(route))
}

func (h *Handler) listRoutes(c *echo.Context) error {
	routes, err := h.usecase.ListRoutes(c.Request().Context(), c.Param("service_id"))
	if err != nil {
		return h.handleError(c, "failed to list routes", err)
	}

	return c.JSON(http.StatusOK, newRouteListResponse(routes))
}
