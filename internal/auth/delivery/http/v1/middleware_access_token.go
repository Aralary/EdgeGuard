package httpdelivery

import (
	"strings"

	"github.com/aralary/edgeguard/internal/auth/usecase"
	"github.com/labstack/echo/v5"
)

func (h *Handler) requireAccessToken(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		rawToken, ok := bearerToken(c.Request().Header.Get("Authorization"))
		if !ok {
			return h.handleError(c, "authenticate access token", usecase.ErrInvalidAccessToken)
		}

		if _, err := h.usecase.AuthenticateAccessToken(c.Request().Context(), rawToken); err != nil {
			return h.handleError(c, "authenticate access token", err)
		}

		return next(c)
	}
}

func bearerToken(header string) (string, bool) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}

	token := strings.TrimSpace(parts[1])
	return token, token != ""
}
