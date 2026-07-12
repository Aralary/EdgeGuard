package httpdelivery

import (
	"net/http"

	"github.com/aralary/edgeguard/internal/auth/usecase"
	"github.com/labstack/echo/v5"
)

func (h *Handler) login(c *echo.Context) error {
	var request loginRequest
	if err := decodeRequest(c, &request); err != nil {
		return h.handleError(c, "failed to decode login request", err)
	}

	session, err := h.usecase.Login(c.Request().Context(), usecase.LoginInput{
		Email:    request.Email,
		Password: request.Password,
	})
	if err != nil {
		return h.handleError(c, "failed to login user", err)
	}

	return c.JSON(http.StatusOK, newSessionResponse(session))
}
