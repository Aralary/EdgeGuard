package httpdelivery

import (
	"encoding/json"
	"net/http"

	"github.com/aralary/edgeguard/internal/controlplane/usecase"
	"github.com/labstack/echo/v5"
)

func (h *Handler) createService(c *echo.Context) error {
	var request createServiceRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&request); err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid request body"})
	}

	service, err := h.usecase.CreateService(c.Request().Context(), usecase.CreateServiceInput{
		ProjectID:   c.Param("project_id"),
		Name:        request.Name,
		UpstreamURL: request.UpstreamURL,
	})
	if err != nil {
		return h.handleError(c, "failed to create service", err)
	}

	return c.JSON(http.StatusCreated, newServiceResponse(service))
}

func (h *Handler) listServices(c *echo.Context) error {
	services, err := h.usecase.ListServices(c.Request().Context(), c.Param("project_id"))
	if err != nil {
		return h.handleError(c, "failed to list services", err)
	}

	return c.JSON(http.StatusOK, newServiceListResponse(services))
}
