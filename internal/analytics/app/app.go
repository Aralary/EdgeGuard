package app

import (
	"context"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/aralary/edgeguard/internal/analytics/config"
	httpdelivery "github.com/aralary/edgeguard/internal/analytics/delivery/http/v1"
	analyticskafka "github.com/aralary/edgeguard/internal/analytics/infrastructure/kafka"
	analyticspostgres "github.com/aralary/edgeguard/internal/analytics/infrastructure/postgres"
	"github.com/aralary/edgeguard/internal/analytics/usecase"
	"github.com/aralary/edgeguard/internal/platform/logger"
	"github.com/aralary/edgeguard/internal/platform/observability"
	platformpostgres "github.com/aralary/edgeguard/internal/platform/postgres"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

func Run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load analytics config: %w", err)
	}

	log := logger.New()

	signalCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	runCtx, cancelRun := context.WithCancel(signalCtx)
	defer cancelRun()

	poolCtx, cancelPool := context.WithTimeout(runCtx, 10*time.Second)
	pool, err := platformpostgres.NewPool(poolCtx, platformpostgres.NewConfigFromEnv())
	cancelPool()
	if err != nil {
		return fmt.Errorf("connect analytics service to postgres: %w", err)
	}
	defer pool.Close()

	repository := analyticspostgres.New(pool)
	analyticsUsecase := usecase.New(usecase.Dependencies{
		GatewayAccessRepository: repository,
		GatewayStatsRepository:  repository,
	})

	consumer, err := analyticskafka.New(
		runCtx,
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

	metrics := observability.NewMetrics("analytics")
	readiness := observability.NewReadiness("analytics",
		observability.Check{Name: "postgres", Run: pool.Ping},
		observability.Check{Name: "kafka", Run: consumer.Ping},
	)

	e := echo.New()
	e.Use(middleware.Recover())
	e.Use(metrics.Middleware())
	observability.Register(e, metrics, readiness)
	handler := httpdelivery.NewHandler(analyticsUsecase, log)
	handler.RegisterRoutes(e)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           e,
		ReadHeaderTimeout: 5 * time.Second,
	}

	consumerErrors := make(chan error, 1)
	go func() {
		log.Infof(
			"analytics consumer started: topic=%s consumer_group=%s",
			cfg.KafkaTopic,
			cfg.KafkaConsumerGroup,
		)
		err := consumer.Run(runCtx)
		if runCtx.Err() != nil {
			consumerErrors <- nil
			return
		}
		consumerErrors <- err
	}()

	serverErrors := make(chan error, 1)
	go func() {
		log.Infof("analytics HTTP server started: addr=%s", server.Addr)
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			serverErrors <- err
			return
		}
		serverErrors <- nil
	}()

	var runErr error
	select {
	case <-signalCtx.Done():
		log.Info("analytics service shutting down")
	case err := <-consumerErrors:
		if err != nil {
			runErr = fmt.Errorf("run analytics Kafka consumer: %w", err)
		}
	case err := <-serverErrors:
		if err != nil {
			runErr = fmt.Errorf("serve analytics HTTP: %w", err)
		}
	}

	cancelRun()
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	if err := server.Shutdown(shutdownCtx); err != nil && runErr == nil {
		runErr = fmt.Errorf("shutdown analytics HTTP server: %w", err)
	}

	return runErr
}
