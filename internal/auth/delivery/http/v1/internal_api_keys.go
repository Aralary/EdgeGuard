package httpdelivery

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
)

const apiKeyHeader = "X-API-Key"

func (h *Handler) validateAPIKey(c *echo.Context) error {
	rawAPIKey := strings.TrimSpace(c.Request().Header.Get(apiKeyHeader))
	principal, err := h.usecase.ValidateAPIKey(c.Request().Context(), rawAPIKey)
	if err != nil {
		return h.handleError(c, "validate api key", err)
	}

	return c.JSON(http.StatusOK, apiKeyPrincipalResponse{
		APIKeyID:  principal.APIKeyID,
		ProjectID: principal.ProjectID,
	})
}
