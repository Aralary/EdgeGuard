package rabbitmq

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	platformtracing "github.com/aralary/edgeguard/internal/platform/tracing"
)

type PolicyConfig struct {
	ManagementURL string
	Username      string
	Password      string
	VHost         string
	Name          string
	Queue         string
	DeadExchange  string
	RetryMin      time.Duration
	RetryMax      time.Duration
}

type PolicyInstaller struct {
	config PolicyConfig
	client *http.Client
}

func NewPolicyInstaller(config PolicyConfig) *PolicyInstaller {
	return &PolicyInstaller{
		config: config,
		client: &http.Client{Timeout: 5 * time.Second, Transport: platformtracing.HTTPTransport(nil)},
	}
}

func (installer *PolicyInstaller) Install(ctx context.Context) error {
	pattern := "^" + regexpQuote(installer.config.Queue) + "$"
	payload := map[string]any{
		"pattern": pattern,
		"definition": map[string]any{
			"dead-letter-exchange": installer.config.DeadExchange,
			"dead-letter-strategy": "at-least-once",
			"overflow":             "reject-publish",
			"delayed-retry-type":   "failed",
			"delayed-retry-min":    installer.config.RetryMin.Milliseconds(),
			"delayed-retry-max":    installer.config.RetryMax.Milliseconds(),
		},
		"priority": 10,
		"apply-to": "quorum_queues",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal RabbitMQ policy: %w", err)
	}

	endpoint := strings.TrimRight(installer.config.ManagementURL, "/") +
		"/api/policies/" + url.PathEscape(installer.config.VHost) + "/" + url.PathEscape(installer.config.Name)
	request, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create RabbitMQ policy request: %w", err)
	}
	request.SetBasicAuth(installer.config.Username, installer.config.Password)
	request.Header.Set("Content-Type", "application/json")

	response, err := installer.client.Do(request)
	if err != nil {
		return fmt.Errorf("install RabbitMQ policy: %w", err)
	}
	defer response.Body.Close()
	responseBody, _ := io.ReadAll(io.LimitReader(response.Body, 8*1024))
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("install RabbitMQ policy: status=%d body=%s", response.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	return nil
}

func regexpQuote(value string) string {
	replacer := strings.NewReplacer(
		`\`, `\\`,
		`.`, `\.`,
		`+`, `\+`,
		`*`, `\*`,
		`?`, `\?`,
		`(`, `\(`,
		`)`, `\)`,
		`[`, `\[`,
		`]`, `\]`,
		`{`, `\{`,
		`}`, `\}`,
		`^`, `\^`,
		`$`, `\$`,
		`|`, `\|`,
	)
	return replacer.Replace(value)
}
