package httpdelivery

import (
	"errors"
	"net/http"

	"github.com/aralary/edgeguard/internal/gateway/domain"
	"github.com/labstack/echo/v5"
)

const apiKeyHeader = "X-API-Key"

func (h *Handler) health(c *echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "gateway",
	})
}

func (h *Handler) proxyRequest(c *echo.Context) error {
	req := c.Request()

	route, err := h.usecase.ResolveRoute(req.Context(), req.URL.Path)
	if err != nil {
		if errors.Is(err, domain.ErrRouteNotFound) {
			return c.String(http.StatusNotFound, "route not found\n")
		}

		h.log.Errorf("failed to resolve route: %v", err)
		return c.String(http.StatusInternalServerError, "internal error\n")
	}

	if err := h.usecase.AuthorizeRoute(req.Context(), route, req.Header.Get(apiKeyHeader)); err != nil {
		switch {
		case errors.Is(err, domain.ErrAPIKeyRequired), errors.Is(err, domain.ErrInvalidAPIKey):
			return c.String(http.StatusUnauthorized, "unauthorized\n")
		case errors.Is(err, domain.ErrAPIKeyProjectMismatch):
			return c.String(http.StatusForbidden, "forbidden\n")
		case errors.Is(err, domain.ErrAuthServiceUnavailable),
			errors.Is(err, domain.ErrAPIKeyValidatorNotConfigured):
			h.log.Errorf("route authorization service unavailable: %v", err)
			return c.String(http.StatusServiceUnavailable, "authorization service unavailable\n")
		default:
			h.log.Errorf("failed to authorize route: %v", err)
			return c.String(http.StatusInternalServerError, "internal error\n")
		}
	}

	// X-API-Key is an EdgeGuard credential and must never be forwarded upstream.
	req.Header.Del(apiKeyHeader)

	h.proxy.ServeHTTP(c.Response(), req, route)

	return nil
}
