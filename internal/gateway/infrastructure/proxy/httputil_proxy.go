package proxy

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/aralary/edgeguard/internal/gateway/domain"
)

type HTTPUtilProxy struct{}

func NewHTTPUtilProxy() *HTTPUtilProxy {
	return &HTTPUtilProxy{}
}

func (p *HTTPUtilProxy) ServeHTTP(w http.ResponseWriter, r *http.Request, route domain.Route) {
	target, err := url.Parse(route.UpstreamURL)
	if err != nil {
		http.Error(w, "bad upstream url", http.StatusBadGateway)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), route.Timeout)
	defer cancel()

	proxy := httputil.NewSingleHostReverseProxy(target)

	originalDirector := proxy.Director
	originalPath := r.URL.Path

	proxy.Director = func(req *http.Request) {
		originalDirector(req)

		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = joinPaths(target.Path, route.UpstreamPath(originalPath))
		req.URL.RawQuery = r.URL.RawQuery
		req.Host = target.Host

		req.Header.Set("X-Forwarded-Host", r.Host)
		req.Header.Set("X-Forwarded-Proto", getRequestScheme(r))
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, "upstream error", http.StatusBadGateway)
	}

	proxy.ServeHTTP(w, r.WithContext(ctx))
}

func joinPaths(a string, b string) string {
	if a == "" {
		return b
	}

	if b == "" {
		return a
	}

	aSlash := strings.HasSuffix(a, "/")
	bSlash := strings.HasPrefix(b, "/")

	switch {
	case aSlash && bSlash:
		return a + b[1:]
	case !aSlash && !bSlash:
		return a + "/" + b
	default:
		return a + b
	}
}

func getRequestScheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}

	return "http"
}