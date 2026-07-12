package httpdelivery

import (
	"net/http"

	"github.com/aralary/edgeguard/internal/auth/usecase"
	"github.com/labstack/echo/v5"
)

func (h *Handler) register(c *echo.Context) error {
	var request registerRequest
	if err := decodeRequest(c, &request); err != nil {
		return h.handleError(c, "failed to decode register request", err)
	}

	user, err := h.usecase.Register(c.Request().Context(), usecase.RegisterInput{
		Email:    request.Email,
		Password: request.Password,
	})
	if err != nil {
		return h.handleError(c, "failed to register user", err)
	}

	return c.JSON(http.StatusCreated, newUserResponse(user))
}
