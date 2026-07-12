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
	case errors.Is(err, usecase.ErrInvalidAccessToken):
		return http.StatusUnauthorized, usecase.ErrInvalidAccessToken.Error(), true
	case errors.Is(err, usecase.ErrInvalidAPIKey):
		return http.StatusUnauthorized, usecase.ErrInvalidAPIKey.Error(), true
	case errors.Is(err, domain.ErrInvalidProjectID):
		return http.StatusBadRequest, domain.ErrInvalidProjectID.Error(), true
	case errors.Is(err, domain.ErrInvalidName):
		return http.StatusBadRequest, domain.ErrInvalidName.Error(), true
	case errors.Is(err, domain.ErrInvalidExpiration):
		return http.StatusBadRequest, domain.ErrInvalidExpiration.Error(), true
	case errors.Is(err, usecase.ErrInvalidAPIKeyID):
		return http.StatusBadRequest, usecase.ErrInvalidAPIKeyID.Error(), true
	case errors.Is(err, usecase.ErrAPIKeyNameAlreadyExists):
		return http.StatusConflict, usecase.ErrAPIKeyNameAlreadyExists.Error(), true
	case errors.Is(err, usecase.ErrProjectNotFound):
		return http.StatusNotFound, usecase.ErrProjectNotFound.Error(), true
	case errors.Is(err, usecase.ErrAPIKeyNotFound):
		return http.StatusNotFound, usecase.ErrAPIKeyNotFound.Error(), true
	case errors.Is(err, usecase.ErrUserDisabled):
		return http.StatusForbidden, usecase.ErrUserDisabled.Error(), true
	default:
		return 0, "", false
	}
}
