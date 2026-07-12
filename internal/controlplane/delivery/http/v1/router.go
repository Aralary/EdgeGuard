package httpdelivery

import (
	"net/http"

	"github.com/labstack/echo/v5"
)

func (h *Handler) RegisterRoutes(e *echo.Echo) {
	e.GET("/health", h.health)

	api := e.Group("/api/v1")
	api.POST("/projects", h.createProject)
	api.GET("/projects", h.listProjects)
	api.POST("/projects/:project_id/services", h.createService)
	api.GET("/projects/:project_id/services", h.listServices)
	api.POST("/services/:service_id/routes", h.createRoute)
	api.GET("/services/:service_id/routes", h.listRoutes)

	internal := e.Group("/internal/v1")
	internal.GET("/routes", h.listGatewayRoutes)
}

func (h *Handler) health(c *echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "control-plane",
	})
}
