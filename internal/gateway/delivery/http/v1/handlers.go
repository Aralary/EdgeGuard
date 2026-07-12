package httpdelivery

import (
	"errors"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/aralary/edgeguard/internal/gateway/domain"
	"github.com/labstack/echo/v5"
)

const (
	apiKeyHeader             = "X-API-Key"
	rateLimitLimitHeader     = "X-RateLimit-Limit"
	rateLimitRemainingHeader = "X-RateLimit-Remaining"
	rateLimitResetHeader     = "X-RateLimit-Reset"
)

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

	principal, err := h.usecase.AuthorizeRoute(req.Context(), route, req.Header.Get(apiKeyHeader))
	if err != nil {
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

	clientID := rateLimitClientID(req, route, principal)
	result, rateLimitErr := h.usecase.CheckRateLimit(req.Context(), route, clientID)
	if rateLimitErr != nil {
		if errors.Is(rateLimitErr, domain.ErrInvalidRateLimitPolicy) ||
			errors.Is(rateLimitErr, domain.ErrInvalidRateLimitClient) {
			h.log.Errorf("invalid route rate limit configuration: route=%s error=%v", route.Name, rateLimitErr)
			return c.String(http.StatusInternalServerError, "internal error\n")
		}

		// Rate limiting is intentionally fail-open: Redis outages must not stop proxy traffic.
		h.log.Warnf(
			"rate limit check failed, allowing request: route=%s client=%s error=%v",
			route.Name,
			clientID,
			rateLimitErr,
		)
	} else if result.Enabled {
		setRateLimitHeaders(c.Response().Header(), result)
		if !result.Allowed {
			return c.String(http.StatusTooManyRequests, "rate limit exceeded\n")
		}
	}

	// X-API-Key is an EdgeGuard credential and must never be forwarded upstream.
	req.Header.Del(apiKeyHeader)

	h.proxy.ServeHTTP(c.Response(), req, route)

	return nil
}

func rateLimitClientID(
	req *http.Request,
	route domain.Route,
	principal domain.APIKeyPrincipal,
) string {
	if route.AuthRequired && strings.TrimSpace(principal.APIKeyID) != "" {
		return "api-key:" + principal.APIKeyID
	}

	return "ip:" + directClientIP(req)
}

func directClientIP(req *http.Request) string {
	remoteAddr := strings.TrimSpace(req.RemoteAddr)
	if remoteAddr == "" {
		return "unknown"
	}

	host, _, err := net.SplitHostPort(remoteAddr)
	if err == nil && host != "" {
		return host
	}

	return remoteAddr
}

func setRateLimitHeaders(header http.Header, result domain.RateLimitResult) {
	header.Set(rateLimitLimitHeader, strconv.Itoa(result.Limit))
	header.Set(rateLimitRemainingHeader, strconv.Itoa(result.Remaining))
	header.Set(rateLimitResetHeader, strconv.FormatInt(result.ResetAt.Unix(), 10))

	if !result.Allowed {
		retryAfterSeconds := int(math.Ceil(result.RetryAfter.Seconds()))
		if retryAfterSeconds < 1 {
			retryAfterSeconds = 1
		}

		header.Set("Retry-After", strconv.Itoa(retryAfterSeconds))
	}
}
