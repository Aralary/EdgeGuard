package postgres

import (
	"os"
	"strconv"
	"time"
)

const defaultDSN = "postgres://edgeguard:edgeguard@localhost:5432/edgeguard?sslmode=disable"

type Config struct {
	DSN             string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
}

func NewConfigFromEnv() Config {
	return Config{
		DSN:             getEnv("POSTGRES_DSN", defaultDSN),
		MaxConns:        int32(getEnvInt("POSTGRES_MAX_CONNS", 10)),
		MinConns:        int32(getEnvInt("POSTGRES_MIN_CONNS", 1)),
		MaxConnLifetime: time.Duration(getEnvInt("POSTGRES_MAX_CONN_LIFETIME_SECONDS", 300)) * time.Second,
	}
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}

func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}
