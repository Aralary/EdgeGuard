package httpdelivery

import (
	"net/http"

	"github.com/aralary/edgeguard/internal/auth/usecase"
	"github.com/labstack/echo/v5"
)

func (h *Handler) createAPIKey(c *echo.Context) error {
	var request createAPIKeyRequest
	if err := decodeRequest(c, &request); err != nil {
		return h.handleError(c, "decode create api key request", err)
	}

	created, err := h.usecase.CreateAPIKey(c.Request().Context(), usecase.CreateAPIKeyInput{
		ProjectID: c.Param("project_id"),
		Name:      request.Name,
		ExpiresAt: request.ExpiresAt,
	})
	if err != nil {
		return h.handleError(c, "create api key", err)
	}

	return c.JSON(http.StatusCreated, newCreatedAPIKeyResponse(created))
}

func (h *Handler) listAPIKeys(c *echo.Context) error {
	keys, err := h.usecase.ListAPIKeys(c.Request().Context(), c.Param("project_id"))
	if err != nil {
		return h.handleError(c, "list api keys", err)
	}

	response := make([]apiKeyResponse, 0, len(keys))
	for _, key := range keys {
		response = append(response, newAPIKeyResponse(key))
	}

	return c.JSON(http.StatusOK, response)
}

func (h *Handler) revokeAPIKey(c *echo.Context) error {
	if err := h.usecase.RevokeAPIKey(
		c.Request().Context(),
		c.Param("project_id"),
		c.Param("api_key_id"),
	); err != nil {
		return h.handleError(c, "revoke api key", err)
	}

	return c.NoContent(http.StatusNoContent)
}
