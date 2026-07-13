package httpdelivery

import (
	"errors"
	"net/http"

	"github.com/aralary/edgeguard/internal/jobs/domain"
	"github.com/labstack/echo/v5"
)

func (h *Handler) handleError(c *echo.Context, operation string, err error) error {
	if isValidationError(err) {
		return c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
	}

	h.log.Errorf("%s: %v", operation, err)
	return c.JSON(http.StatusServiceUnavailable, errorResponse{Error: "job queue is unavailable"})
}

func isValidationError(err error) bool {
	return errors.Is(err, errInvalidRequestBody) ||
		errors.Is(err, domain.ErrInvalidWebhookURL) ||
		errors.Is(err, domain.ErrInvalidHTTPMethod) ||
		errors.Is(err, domain.ErrInvalidWebhookBody) ||
		errors.Is(err, domain.ErrInvalidProjectID) ||
		errors.Is(err, domain.ErrInvalidReportRange) ||
		errors.Is(err, domain.ErrInvalidReportFormat) ||
		errors.Is(err, domain.ErrInvalidCleanupTime) ||
		errors.Is(err, domain.ErrInvalidBatchSize) ||
		errors.Is(err, domain.ErrInvalidMaxAttempts)
}
