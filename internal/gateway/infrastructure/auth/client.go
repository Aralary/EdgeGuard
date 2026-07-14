package auth

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
	validateAPIKeyPath = "internal/v1/api-keys/validate"
	apiKeyHeader       = "X-API-Key"
	defaultHTTPTimeout = 3 * time.Second
	maxResponseSize    = 64 << 10
)

type Client struct {
	validateURL string
	httpClient  *http.Client
}

type validateAPIKeyResponse struct {
	APIKeyID  string `json:"api_key_id"`
	ProjectID string `json:"project_id"`
}

func New(baseURL string, httpClient *http.Client) (*Client, error) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return nil, errors.New("auth service base url is required")
	}

	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse auth service base url: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, errors.New("auth service base url must use http or https")
	}

	if parsedURL.Host == "" {
		return nil, errors.New("auth service base url host is required")
	}

	validateURL, err := url.JoinPath(parsedURL.String(), validateAPIKeyPath)
	if err != nil {
		return nil, fmt.Errorf("build api key validation url: %w", err)
	}

	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultHTTPTimeout, Transport: platformtracing.HTTPTransport(nil)}
	}

	return &Client{
		validateURL: validateURL,
		httpClient:  httpClient,
	}, nil
}

func (c *Client) ValidateAPIKey(
	ctx context.Context,
	rawAPIKey string,
) (domain.APIKeyPrincipal, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.validateURL, nil)
	if err != nil {
		return domain.APIKeyPrincipal{}, fmt.Errorf("create api key validation request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set(apiKeyHeader, rawAPIKey)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return domain.APIKeyPrincipal{}, fmt.Errorf("%w: request api key validation: %w", domain.ErrAuthServiceUnavailable, err)
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusUnauthorized {
		_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, maxResponseSize))
		return domain.APIKeyPrincipal{}, domain.ErrInvalidAPIKey
	}

	if res.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(io.LimitReader(res.Body, maxResponseSize))
		if readErr != nil {
			return domain.APIKeyPrincipal{}, fmt.Errorf(
				"%w: auth service returned status %d and body could not be read: %v",
				domain.ErrAuthServiceUnavailable,
				res.StatusCode,
				readErr,
			)
		}

		message := strings.TrimSpace(string(body))
		if message == "" {
			return domain.APIKeyPrincipal{}, fmt.Errorf(
				"%w: auth service returned status %d",
				domain.ErrAuthServiceUnavailable,
				res.StatusCode,
			)
		}

		return domain.APIKeyPrincipal{}, fmt.Errorf(
			"%w: auth service returned status %d: %s",
			domain.ErrAuthServiceUnavailable,
			res.StatusCode,
			message,
		)
	}

	var response validateAPIKeyResponse
	decoder := json.NewDecoder(io.LimitReader(res.Body, maxResponseSize))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&response); err != nil {
		return domain.APIKeyPrincipal{}, fmt.Errorf(
			"%w: decode api key validation response: %v",
			domain.ErrAuthServiceUnavailable,
			err,
		)
	}

	if err := ensureJSONEOF(decoder); err != nil {
		return domain.APIKeyPrincipal{}, fmt.Errorf("%w: %v", domain.ErrAuthServiceUnavailable, err)
	}

	response.APIKeyID = strings.TrimSpace(response.APIKeyID)
	response.ProjectID = strings.TrimSpace(response.ProjectID)
	if response.APIKeyID == "" || response.ProjectID == "" {
		return domain.APIKeyPrincipal{}, fmt.Errorf(
			"%w: api key validation response is missing principal fields",
			domain.ErrAuthServiceUnavailable,
		)
	}

	return domain.APIKeyPrincipal{
		APIKeyID:  response.APIKeyID,
		ProjectID: response.ProjectID,
	}, nil
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); errors.Is(err, io.EOF) {
		return nil
	} else if err != nil {
		return fmt.Errorf("decode trailing api key validation response: %w", err)
	}

	return errors.New("api key validation response contains multiple json values")
}
