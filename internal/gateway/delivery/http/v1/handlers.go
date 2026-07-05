package httpdelivery

import (
	"errors"
	"net/http"

	"github.com/aralary/edgeguard/internal/gateway/domain"
	"github.com/labstack/echo/v5"
)

func (h *Handler) proxyRequest(c *echo.Context) error {
	req := c.Request()

	route, err := h.resolveRoute.Execute(req.Context(), req.URL.Path)
	if err != nil {
		if errors.Is(err, domain.ErrRouteNotFound) {
			return c.String(http.StatusNotFound, "route not found\n")
		}

		h.log.WithError(err).Error("failed to resolve route")
		return c.String(http.StatusInternalServerError, "internal error")
	}

	h.proxy.ServeHTTP(c.Response(), req, route)

	return nil
}
