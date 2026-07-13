package app

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"
	"time"

	"github.com/aralary/edgeguard/internal/analytics/config"
	analyticskafka "github.com/aralary/edgeguard/internal/analytics/infrastructure/kafka"
	analyticspostgres "github.com/aralary/edgeguard/internal/analytics/infrastructure/postgres"
	"github.com/aralary/edgeguard/internal/analytics/usecase"
	"github.com/aralary/edgeguard/internal/platform/logger"
	platformpostgres "github.com/aralary/edgeguard/internal/platform/postgres"
)

func Run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load analytics config: %w", err)
	}

	log := logger.New()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	poolCtx, cancelPool := context.WithTimeout(ctx, 10*time.Second)
	pool, err := platformpostgres.NewPool(poolCtx, platformpostgres.NewConfigFromEnv())
	cancelPool()
	if err != nil {
		return fmt.Errorf("connect analytics worker to postgres: %w", err)
	}
	defer pool.Close()

	repository := analyticspostgres.New(pool)
	analyticsUsecase := usecase.New(usecase.Dependencies{
		GatewayAccessRepository: repository,
	})

	consumer, err := analyticskafka.New(
		ctx,
		analyticskafka.Config{
			Brokers:            cfg.KafkaBrokers,
			Topic:              cfg.KafkaTopic,
			ConsumerGroup:      cfg.KafkaConsumerGroup,
			ClientID:           cfg.KafkaClientID,
			RetryMin:           cfg.RetryMin,
			RetryMax:           cfg.RetryMax,
			MaxProcessAttempts: cfg.MaxProcessAttempts,
		},
		analyticsUsecase,
		log,
	)
	if err != nil {
		return fmt.Errorf("create analytics Kafka consumer: %w", err)
	}
	defer consumer.Close()

	log.Infof(
		"analytics worker started: topic=%s consumer_group=%s",
		cfg.KafkaTopic,
		cfg.KafkaConsumerGroup,
	)

	if err := consumer.Run(ctx); err != nil {
		return fmt.Errorf("run analytics Kafka consumer: %w", err)
	}

	log.Info("analytics worker stopped")
	return nil
}
