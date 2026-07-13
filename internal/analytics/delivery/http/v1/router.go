package httpdelivery

import (
	"net/http"

	"github.com/labstack/echo/v5"
)

func (h *Handler) RegisterRoutes(e *echo.Echo) {
	e.GET("/health", h.health)

	analytics := e.Group("/api/v1/projects/:project_id/analytics")
	analytics.GET("/summary", h.summary)
	analytics.GET("/hourly", h.hourly)
}

func (h *Handler) health(c *echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "analytics",
	})
}
