package tracing

import "testing"

func TestLoadConfigDefaults(t *testing.T) {
	t.Setenv("OTEL_SERVICE_NAME", "")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	t.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "")
	t.Setenv("OTEL_DEPLOYMENT_ENVIRONMENT", "")
	t.Setenv("OTEL_TRACES_SAMPLER_RATIO", "")

	config, err := LoadConfig("gateway")
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if config.ServiceName != "gateway" || config.Environment != "local" || config.SampleRatio != 1 {
		t.Fatalf("unexpected config: %#v", config)
	}
	if !config.Insecure {
		t.Fatal("Insecure = false, want true")
	}
}

func TestLoadConfigRejectsInvalidRatio(t *testing.T) {
	t.Setenv("OTEL_TRACES_SAMPLER_RATIO", "1.1")
	if _, err := LoadConfig("gateway"); err == nil {
		t.Fatal("LoadConfig() error = nil, want error")
	}
}
