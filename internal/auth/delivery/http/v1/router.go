package httpdelivery

import (
	"net/http"

	"github.com/labstack/echo/v5"
)

func (h *Handler) RegisterRoutes(e *echo.Echo) {
	e.GET("/health", h.health)

	auth := e.Group("/api/v1/auth")
	auth.POST("/register", h.register)
	auth.POST("/login", h.login)
	auth.POST("/refresh", h.refresh)
	auth.POST("/logout", h.logout)
}

func (h *Handler) health(c *echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "auth",
	})
}
