package httpdelivery

import "github.com/labstack/echo/v5"

func (h *Handler) RegisterRoutes(e *echo.Echo) {
	jobs := e.Group("/api/v1/jobs")
	jobs.POST("/webhooks", h.submitWebhook)
	jobs.POST("/reports", h.submitReport)
	jobs.POST("/cleanup", h.submitCleanup)
	jobs.GET("/:job_id", h.getJob)
}
