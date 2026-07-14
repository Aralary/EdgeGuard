package webhook

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aralary/edgeguard/internal/jobworker/domain"
	platformjobs "github.com/aralary/edgeguard/internal/platform/jobs"
	platformtracing "github.com/aralary/edgeguard/internal/platform/tracing"
)

const maxResponseBodyBytes = 64 * 1024

var (
	ErrInvalidURL         = errors.New("invalid webhook url")
	ErrInvalidMethod      = errors.New("invalid webhook method")
	ErrBlockedTarget      = errors.New("webhook target is blocked")
	ErrUnexpectedStatus   = errors.New("unexpected webhook response status")
	ErrNonRetryableStatus = errors.New("non-retryable webhook response status")
)

type Config struct {
	Timeout      time.Duration
	AllowedHosts []string
}

type Sender struct {
	timeout      time.Duration
	allowedHosts map[string]struct{}
	resolver     *net.Resolver
}

func New(config Config) *Sender {
	allowedHosts := make(map[string]struct{}, len(config.AllowedHosts))
	for _, host := range config.AllowedHosts {
		host = strings.ToLower(strings.TrimSpace(host))
		if host != "" {
			allowedHosts[host] = struct{}{}
		}
	}

	return &Sender{
		timeout:      config.Timeout,
		allowedHosts: allowedHosts,
		resolver:     net.DefaultResolver,
	}
}

func (s *Sender) Deliver(ctx context.Context, payload platformjobs.WebhookPayload) error {
	target, addresses, err := s.resolveTarget(ctx, payload.URL)
	if err != nil {
		return err
	}

	method := strings.ToUpper(strings.TrimSpace(payload.Method))
	if method != http.MethodPost && method != http.MethodPut && method != http.MethodPatch {
		return domain.Permanent(ErrInvalidMethod)
	}

	request, err := http.NewRequestWithContext(ctx, method, target.String(), bytes.NewReader(payload.Body))
	if err != nil {
		return domain.Permanent(fmt.Errorf("create webhook request: %w", err))
	}
	for name, value := range payload.Headers {
		request.Header.Set(name, value)
	}
	if len(payload.Body) > 0 && request.Header.Get("Content-Type") == "" {
		request.Header.Set("Content-Type", "application/json")
	}
	request.Header.Set("User-Agent", "EdgeGuard-Notification-Worker/1.0")

	transport := &http.Transport{
		Proxy:               nil,
		DisableKeepAlives:   true,
		ForceAttemptHTTP2:   true,
		TLSHandshakeTimeout: 5 * time.Second,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: target.Hostname(),
		},
		DialContext: fixedDialer(target.Hostname(), addresses),
	}
	client := &http.Client{
		Transport: platformtracing.HTTPTransport(transport),
		Timeout:   s.timeout,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	defer transport.CloseIdleConnections()

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("deliver webhook: %w", err)
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, maxResponseBodyBytes))

	if response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices {
		return nil
	}
	if retryableStatus(response.StatusCode) {
		return fmt.Errorf("%w: status=%d", ErrUnexpectedStatus, response.StatusCode)
	}

	return domain.Permanent(fmt.Errorf("%w: status=%d", ErrNonRetryableStatus, response.StatusCode))
}

func (s *Sender) resolveTarget(ctx context.Context, rawURL string) (*url.URL, []net.IP, error) {
	target, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || target.Hostname() == "" || (target.Scheme != "http" && target.Scheme != "https") || target.User != nil {
		return nil, nil, domain.Permanent(ErrInvalidURL)
	}

	host := strings.ToLower(target.Hostname())
	_, explicitlyAllowed := s.allowedHosts[host]

	var addresses []net.IP
	if parsed := net.ParseIP(host); parsed != nil {
		addresses = []net.IP{parsed}
	} else {
		resolved, err := s.resolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, nil, fmt.Errorf("resolve webhook host: %w", err)
		}
		addresses = make([]net.IP, 0, len(resolved))
		for _, address := range resolved {
			addresses = append(addresses, address.IP)
		}
	}
	if len(addresses) == 0 {
		return nil, nil, fmt.Errorf("resolve webhook host: no addresses")
	}

	if !explicitlyAllowed {
		for _, address := range addresses {
			if blockedIP(address) {
				return nil, nil, domain.Permanent(fmt.Errorf("%w: %s", ErrBlockedTarget, address))
			}
		}
	}

	return target, addresses, nil
}

func fixedDialer(expectedHost string, addresses []net.IP) func(context.Context, string, string) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}

	return func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		if !strings.EqualFold(host, expectedHost) {
			return nil, domain.Permanent(fmt.Errorf("%w: unexpected redirect host %q", ErrBlockedTarget, host))
		}

		var lastError error
		for _, ip := range addresses {
			connection, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
			if err == nil {
				return connection, nil
			}
			lastError = err
		}

		return nil, fmt.Errorf("dial webhook target: %w", lastError)
	}
}

func blockedIP(ip net.IP) bool {
	return ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsUnspecified() ||
		ip.IsMulticast() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast()
}

func retryableStatus(statusCode int) bool {
	return statusCode == http.StatusRequestTimeout ||
		statusCode == http.StatusTooEarly ||
		statusCode == http.StatusTooManyRequests ||
		statusCode >= http.StatusInternalServerError
}
