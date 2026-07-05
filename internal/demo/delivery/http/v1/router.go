package httpdelivery

import "github.com/labstack/echo/v5"

func (h *Handler) RegisterRoutes(e *echo.Echo) {
	e.GET("/health", h.health)
	e.GET("/orders", h.list)
	e.GET("/orders/:id", h.get)
	e.POST("/orders", h.create)
}
