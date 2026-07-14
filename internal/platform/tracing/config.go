package tracing

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	defaultEnvironment = "local"
	defaultSampleRatio = 1.0
)

type Config struct {
	ServiceName string
	Endpoint    string
	Insecure    bool
	Environment string
	SampleRatio float64
}

func LoadConfig(serviceName string) (Config, error) {
	serviceName = strings.TrimSpace(serviceName)
	if override := strings.TrimSpace(os.Getenv("OTEL_SERVICE_NAME")); override != "" {
		serviceName = override
	}
	if serviceName == "" {
		return Config{}, fmt.Errorf("OpenTelemetry service name is required")
	}

	insecure, err := boolFromEnv("OTEL_EXPORTER_OTLP_INSECURE", true)
	if err != nil {
		return Config{}, err
	}

	sampleRatio, err := ratioFromEnv("OTEL_TRACES_SAMPLER_RATIO", defaultSampleRatio)
	if err != nil {
		return Config{}, err
	}

	return Config{
		ServiceName: serviceName,
		Endpoint:    strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")),
		Insecure:    insecure,
		Environment: stringFromEnv("OTEL_DEPLOYMENT_ENVIRONMENT", defaultEnvironment),
		SampleRatio: sampleRatio,
	}, nil
}

func stringFromEnv(name string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}

func boolFromEnv(name string, fallback bool) (bool, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("parse %s: %w", name, err)
	}
	return parsed, nil
}

func ratioFromEnv(name string, fallback float64) (float64, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}
	if parsed < 0 || parsed > 1 {
		return 0, fmt.Errorf("%s must be between 0 and 1", name)
	}
	return parsed, nil
}
