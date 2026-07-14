package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultRabbitMQURL          = "amqp://guest:guest@localhost:5672/"
	defaultManagementURL        = "http://localhost:15672"
	defaultRabbitMQUsername     = "guest"
	defaultRabbitMQPassword     = "guest"
	defaultRabbitMQVHost        = "/"
	defaultExchange             = "edgeguard.jobs.v1"
	defaultQueue                = "edgeguard.jobs.main.v1"
	defaultDeadExchange         = "edgeguard.jobs.dead.v1"
	defaultDeadQueue            = "edgeguard.jobs.dead.v1"
	defaultPolicyName           = "edgeguard-jobs-main"
	defaultPrefetch             = 10
	defaultRetryMin             = 5 * time.Second
	defaultRetryMax             = time.Minute
	defaultProcessTimeout       = 2 * time.Minute
	defaultWebhookTimeout       = 15 * time.Second
	defaultReportDirectory      = "./data/reports"
	defaultPolicyInstallTimeout = 30 * time.Second
	defaultObservabilityAddr    = ":8085"
)

var (
	ErrInvalidPrefetch             = errors.New("invalid jobs worker prefetch")
	ErrInvalidRetryMin             = errors.New("invalid jobs retry min")
	ErrInvalidRetryMax             = errors.New("invalid jobs retry max")
	ErrInvalidProcessTimeout       = errors.New("invalid jobs process timeout")
	ErrInvalidWebhookTimeout       = errors.New("invalid jobs webhook timeout")
	ErrInvalidPolicyInstallTimeout = errors.New("invalid RabbitMQ policy install timeout")
)

type Config struct {
	RabbitMQURL          string
	ManagementURL        string
	RabbitMQUsername     string
	RabbitMQPassword     string
	RabbitMQVHost        string
	Exchange             string
	Queue                string
	DeadExchange         string
	DeadQueue            string
	PolicyName           string
	Prefetch             int
	RetryMin             time.Duration
	RetryMax             time.Duration
	ProcessTimeout       time.Duration
	WebhookTimeout       time.Duration
	WebhookAllowedHosts  []string
	ReportDirectory      string
	PolicyInstallTimeout time.Duration
	ObservabilityAddr    string
}

func Load() (Config, error) {
	prefetch, err := positiveIntEnv("JOBS_WORKER_PREFETCH", defaultPrefetch)
	if err != nil {
		return Config{}, ErrInvalidPrefetch
	}

	retryMin, err := positiveDurationEnv("JOBS_RETRY_MIN", defaultRetryMin)
	if err != nil {
		return Config{}, ErrInvalidRetryMin
	}
	retryMax, err := positiveDurationEnv("JOBS_RETRY_MAX", defaultRetryMax)
	if err != nil || retryMax < retryMin {
		return Config{}, ErrInvalidRetryMax
	}

	processTimeout, err := positiveDurationEnv("JOBS_PROCESS_TIMEOUT", defaultProcessTimeout)
	if err != nil {
		return Config{}, ErrInvalidProcessTimeout
	}
	webhookTimeout, err := positiveDurationEnv("JOBS_WEBHOOK_TIMEOUT", defaultWebhookTimeout)
	if err != nil {
		return Config{}, ErrInvalidWebhookTimeout
	}
	policyInstallTimeout, err := positiveDurationEnv("JOBS_POLICY_INSTALL_TIMEOUT", defaultPolicyInstallTimeout)
	if err != nil {
		return Config{}, ErrInvalidPolicyInstallTimeout
	}

	return Config{
		RabbitMQURL:          envOrDefault("RABBITMQ_URL", defaultRabbitMQURL),
		ManagementURL:        strings.TrimRight(envOrDefault("RABBITMQ_MANAGEMENT_URL", defaultManagementURL), "/"),
		RabbitMQUsername:     envOrDefault("RABBITMQ_USERNAME", defaultRabbitMQUsername),
		RabbitMQPassword:     envOrDefault("RABBITMQ_PASSWORD", defaultRabbitMQPassword),
		RabbitMQVHost:        envOrDefault("RABBITMQ_VHOST", defaultRabbitMQVHost),
		Exchange:             envOrDefault("JOBS_EXCHANGE", defaultExchange),
		Queue:                envOrDefault("JOBS_QUEUE", defaultQueue),
		DeadExchange:         envOrDefault("JOBS_DEAD_EXCHANGE", defaultDeadExchange),
		DeadQueue:            envOrDefault("JOBS_DEAD_QUEUE", defaultDeadQueue),
		PolicyName:           envOrDefault("JOBS_POLICY_NAME", defaultPolicyName),
		Prefetch:             prefetch,
		RetryMin:             retryMin,
		RetryMax:             retryMax,
		ProcessTimeout:       processTimeout,
		WebhookTimeout:       webhookTimeout,
		WebhookAllowedHosts:  csvEnv("JOBS_WEBHOOK_ALLOWED_HOSTS"),
		ReportDirectory:      envOrDefault("JOBS_REPORT_DIRECTORY", defaultReportDirectory),
		PolicyInstallTimeout: policyInstallTimeout,
		ObservabilityAddr:    envOrDefault("JOBS_OBSERVABILITY_ADDR", defaultObservabilityAddr),
	}, nil
}

func envOrDefault(name, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}

	return value
}

func positiveIntEnv(name string, fallback int) (int, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 0, errors.New("value must be a positive integer")
	}

	return parsed, nil
}

func positiveDurationEnv(name string, fallback time.Duration) (time.Duration, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}

	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return 0, errors.New("value must be a positive duration")
	}

	return parsed, nil
}

func csvEnv(name string) []string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.ToLower(strings.TrimSpace(part))
		if part != "" {
			result = append(result, part)
		}
	}

	return result
}
