package httpdelivery

import (
	"net/http"

	"github.com/labstack/echo/v5"
)

func (h *Handler) listGatewayRoutes(c *echo.Context) error {
	routes, err := h.usecase.ListGatewayRoutes(c.Request().Context())
	if err != nil {
		return h.handleError(c, "failed to list gateway routes", err)
	}

	return c.JSON(http.StatusOK, newGatewayRouteListResponse(routes))
}
