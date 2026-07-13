package httpdelivery

import (
	"net/http"

	"github.com/labstack/echo/v5"
)

func (h *Handler) getJob(c *echo.Context) error {
	job, err := h.usecase.GetJob(c.Request().Context(), c.Param("job_id"))
	if err != nil {
		return h.handleGetError(c, "get background job", err)
	}

	return c.JSON(http.StatusOK, newJobStatusResponse(job))
}
