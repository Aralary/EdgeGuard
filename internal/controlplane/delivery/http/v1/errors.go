package httpdelivery

import (
	"errors"
	"net/http"

	"github.com/aralary/edgeguard/internal/controlplane/domain"
	"github.com/labstack/echo/v5"
)

func (h *Handler) handleError(c *echo.Context, operation string, err error) error {
	if isValidationError(err) {
		return c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
	}

	h.log.Errorf("%s: %v", operation, err)
	return c.JSON(http.StatusInternalServerError, errorResponse{Error: "internal error"})
}

func isValidationError(err error) bool {
	return errors.Is(err, domain.ErrInvalidName) ||
		errors.Is(err, domain.ErrInvalidProjectID) ||
		errors.Is(err, domain.ErrInvalidServiceID) ||
		errors.Is(err, domain.ErrInvalidUpstreamURL) ||
		errors.Is(err, domain.ErrInvalidPathPrefix) ||
		errors.Is(err, domain.ErrInvalidRateLimitRequests) ||
		errors.Is(err, domain.ErrInvalidRateLimitWindow)
}
