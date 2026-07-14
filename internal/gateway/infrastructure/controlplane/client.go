package controlplane

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aralary/edgeguard/internal/gateway/domain"
	platformtracing "github.com/aralary/edgeguard/internal/platform/tracing"
)

const (
	routesPath         = "internal/v1/routes"
	defaultHTTPTimeout = 5 * time.Second
	maxResponseSize    = 1 << 20 // 1 MiB
)

type Client struct {
	routesURL  string
	httpClient *http.Client
}

func New(baseURL string, httpClient *http.Client) (*Client, error) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return nil, errors.New("control plane base url is required")
	}

	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse control plane base url: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, errors.New("control plane base url must use http or https")
	}

	if parsedURL.Host == "" {
		return nil, errors.New("control plane base url host is required")
	}

	routesURL, err := url.JoinPath(parsedURL.String(), routesPath)
	if err != nil {
		return nil, fmt.Errorf("build control plane routes url: %w", err)
	}

	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultHTTPTimeout, Transport: platformtracing.HTTPTransport(nil)}
	}

	return &Client{
		routesURL:  routesURL,
		httpClient: httpClient,
	}, nil
}

func (c *Client) ListRoutes(ctx context.Context) ([]domain.Route, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.routesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create routes request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request control plane routes: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(io.LimitReader(res.Body, maxResponseSize))
		if readErr != nil {
			return nil, fmt.Errorf("control plane routes returned status %d and response body could not be read: %w", res.StatusCode, readErr)
		}

		message := strings.TrimSpace(string(body))
		if message == "" {
			return nil, fmt.Errorf("control plane routes returned status %d", res.StatusCode)
		}

		return nil, fmt.Errorf("control plane routes returned status %d: %s", res.StatusCode, message)
	}

	var response []routeResponse
	decoder := json.NewDecoder(io.LimitReader(res.Body, maxResponseSize))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&response); err != nil {
		return nil, fmt.Errorf("decode control plane routes response: %w", err)
	}

	if err := ensureJSONEOF(decoder); err != nil {
		return nil, err
	}

	routes, err := responseToDomain(response)
	if err != nil {
		return nil, err
	}

	return routes, nil
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err == io.EOF {
		return nil
	} else if err != nil {
		return fmt.Errorf("decode trailing control plane routes response: %w", err)
	}

	return errors.New("control plane routes response contains multiple json values")
}
