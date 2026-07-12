package httpdelivery

import (
	"net/http"

	"github.com/aralary/edgeguard/internal/auth/usecase"
	"github.com/labstack/echo/v5"
)

func (h *Handler) refresh(c *echo.Context) error {
	var request refreshRequest
	if err := decodeRequest(c, &request); err != nil {
		return h.handleError(c, "failed to decode refresh request", err)
	}

	tokens, err := h.usecase.Refresh(c.Request().Context(), usecase.RefreshInput{
		RefreshToken: request.RefreshToken,
	})
	if err != nil {
		return h.handleError(c, "failed to refresh tokens", err)
	}

	return c.JSON(http.StatusOK, newTokenPairResponse(tokens))
}

func (h *Handler) logout(c *echo.Context) error {
	var request logoutRequest
	if err := decodeRequest(c, &request); err != nil {
		return h.handleError(c, "failed to decode logout request", err)
	}

	if err := h.usecase.Logout(c.Request().Context(), usecase.LogoutInput{
		RefreshToken: request.RefreshToken,
	}); err != nil {
		return h.handleError(c, "failed to logout user", err)
	}

	return c.NoContent(http.StatusNoContent)
}
