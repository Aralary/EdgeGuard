package config

import (
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	t.Setenv("KAFKA_BROKERS", " kafka-1:9092, kafka-2:9092, kafka-1:9092 ")
	t.Setenv("KAFKA_ACCESS_TOPIC", "events")
	t.Setenv("ANALYTICS_KAFKA_GROUP", "group")
	t.Setenv("ANALYTICS_KAFKA_CLIENT_ID", "client")
	t.Setenv("ANALYTICS_RETRY_MIN", "2s")
	t.Setenv("ANALYTICS_RETRY_MAX", "20s")
	t.Setenv("ANALYTICS_MAX_PROCESS_ATTEMPTS", "7")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(cfg.KafkaBrokers) != 2 || cfg.KafkaBrokers[0] != "kafka-1:9092" || cfg.KafkaBrokers[1] != "kafka-2:9092" {
		t.Fatalf("unexpected brokers: %#v", cfg.KafkaBrokers)
	}
	if cfg.KafkaTopic != "events" || cfg.KafkaConsumerGroup != "group" || cfg.KafkaClientID != "client" {
		t.Fatalf("unexpected Kafka config: %#v", cfg)
	}
	if cfg.RetryMin != 2*time.Second || cfg.RetryMax != 20*time.Second {
		t.Fatalf("unexpected retry config: min=%s max=%s", cfg.RetryMin, cfg.RetryMax)
	}
	if cfg.MaxProcessAttempts != 7 {
		t.Fatalf("MaxProcessAttempts = %d, want 7", cfg.MaxProcessAttempts)
	}
}

func TestLoadRequiresKafkaBrokers(t *testing.T) {
	t.Setenv("KAFKA_BROKERS", "")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want error")
	}
}

func TestLoadRejectsRetryMaxBelowRetryMin(t *testing.T) {
	t.Setenv("KAFKA_BROKERS", "kafka:9092")
	t.Setenv("ANALYTICS_RETRY_MIN", "10s")
	t.Setenv("ANALYTICS_RETRY_MAX", "5s")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want error")
	}
}

func TestLoadRejectsNonPositiveMaxProcessAttempts(t *testing.T) {
	t.Setenv("KAFKA_BROKERS", "kafka:9092")
	t.Setenv("ANALYTICS_MAX_PROCESS_ATTEMPTS", "0")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want error")
	}
}
