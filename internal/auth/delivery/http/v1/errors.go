package httpdelivery

import (
	"errors"
	"net/http"

	"github.com/aralary/edgeguard/internal/auth/domain"
	"github.com/aralary/edgeguard/internal/auth/usecase"
	"github.com/labstack/echo/v5"
)

func (h *Handler) handleError(c *echo.Context, operation string, err error) error {
	status, message, expected := authError(err)
	if expected {
		return c.JSON(status, errorResponse{Error: message})
	}

	h.log.Errorf("%s: %v", operation, err)
	return c.JSON(http.StatusInternalServerError, errorResponse{Error: "internal error"})
}

func authError(err error) (status int, message string, expected bool) {
	switch {
	case errors.Is(err, errInvalidRequestBody):
		return http.StatusBadRequest, errInvalidRequestBody.Error(), true
	case errors.Is(err, domain.ErrInvalidEmail):
		return http.StatusBadRequest, domain.ErrInvalidEmail.Error(), true
	case errors.Is(err, usecase.ErrInvalidPassword):
		return http.StatusBadRequest, usecase.ErrInvalidPassword.Error(), true
	case errors.Is(err, usecase.ErrEmailAlreadyExists):
		return http.StatusConflict, usecase.ErrEmailAlreadyExists.Error(), true
	case errors.Is(err, usecase.ErrInvalidCredentials):
		return http.StatusUnauthorized, usecase.ErrInvalidCredentials.Error(), true
	case errors.Is(err, usecase.ErrInvalidRefreshToken):
		return http.StatusUnauthorized, usecase.ErrInvalidRefreshToken.Error(), true
	case errors.Is(err, usecase.ErrUserDisabled):
		return http.StatusForbidden, usecase.ErrUserDisabled.Error(), true
	default:
		return 0, "", false
	}
}
