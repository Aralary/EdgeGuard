package app

import (
	"context"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	jobspostgres "github.com/aralary/edgeguard/internal/jobs/infrastructure/postgres"
	jobworkerconfig "github.com/aralary/edgeguard/internal/jobworker/config"
	jobworkerpostgres "github.com/aralary/edgeguard/internal/jobworker/infrastructure/postgres"
	jobworkerrabbitmq "github.com/aralary/edgeguard/internal/jobworker/infrastructure/rabbitmq"
	jobworkerreport "github.com/aralary/edgeguard/internal/jobworker/infrastructure/report"
	jobworkerwebhook "github.com/aralary/edgeguard/internal/jobworker/infrastructure/webhook"
	jobworkerusecase "github.com/aralary/edgeguard/internal/jobworker/usecase"
	"github.com/aralary/edgeguard/internal/platform/logger"
	"github.com/aralary/edgeguard/internal/platform/observability"
	platformpostgres "github.com/aralary/edgeguard/internal/platform/postgres"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

func Run() error {
	log := logger.New()
	signalCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	runCtx, cancelRun := context.WithCancel(signalCtx)
	defer cancelRun()

	config, err := jobworkerconfig.Load()
	if err != nil {
		return fmt.Errorf("load notification worker config: %w", err)
	}

	poolCtx, cancelPool := context.WithTimeout(runCtx, 10*time.Second)
	pool, err := platformpostgres.NewPool(poolCtx, platformpostgres.NewConfigFromEnv())
	cancelPool()
	if err != nil {
		return fmt.Errorf("connect notification worker to postgres: %w", err)
	}
	defer pool.Close()

	consumer, err := jobworkerrabbitmq.NewConsumer(jobworkerrabbitmq.ConsumerConfig{
		URL:            config.RabbitMQURL,
		Exchange:       config.Exchange,
		Queue:          config.Queue,
		DeadExchange:   config.DeadExchange,
		DeadQueue:      config.DeadQueue,
		Prefetch:       config.Prefetch,
		ProcessTimeout: config.ProcessTimeout,
	})
	if err != nil {
		return fmt.Errorf("create RabbitMQ job consumer: %w", err)
	}
	defer consumer.Close()

	policyInstaller := jobworkerrabbitmq.NewPolicyInstaller(jobworkerrabbitmq.PolicyConfig{
		ManagementURL: config.ManagementURL,
		Username:      config.RabbitMQUsername,
		Password:      config.RabbitMQPassword,
		VHost:         config.RabbitMQVHost,
		Name:          config.PolicyName,
		Queue:         config.Queue,
		DeadExchange:  config.DeadExchange,
		RetryMin:      config.RetryMin,
		RetryMax:      config.RetryMax,
	})
	if err := installPolicy(runCtx, policyInstaller, config.PolicyInstallTimeout, log); err != nil {
		return err
	}

	jobStatusRepository := jobspostgres.New(pool)
	jobUsecase := jobworkerusecase.New(jobworkerusecase.Dependencies{
		WebhookSender: jobworkerwebhook.New(jobworkerwebhook.Config{
			Timeout:      config.WebhookTimeout,
			AllowedHosts: config.WebhookAllowedHosts,
		}),
		ReportGenerator: jobworkerreport.NewGenerator(pool, config.ReportDirectory),
		TokenCleaner:    jobworkerpostgres.NewTokenCleaner(pool),
	})

	metrics := observability.NewMetrics("notification-worker")
	readiness := observability.NewReadiness("notification-worker",
		observability.Check{Name: "postgres", Run: pool.Ping},
		observability.Check{Name: "rabbitmq", Run: consumer.Ping},
	)

	e := echo.New()
	e.Use(middleware.Recover())
	e.Use(metrics.Middleware())
	e.GET("/health", func(c *echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status":  "ok",
			"service": "notification-worker",
		})
	})
	observability.Register(e, metrics, readiness)

	server := &http.Server{
		Addr:              config.ObservabilityAddr,
		Handler:           e,
		ReadHeaderTimeout: 5 * time.Second,
	}

	consumerErrors := make(chan error, 1)
	go func() {
		log.Infof(
			"notification worker started: queue=%s dead_queue=%s prefetch=%d retry=%s..%s",
			config.Queue,
			config.DeadQueue,
			config.Prefetch,
			config.RetryMin,
			config.RetryMax,
		)
		consumerErrors <- consumer.Run(runCtx, jobUsecase, jobStatusRepository, log)
	}()

	serverErrors := make(chan error, 1)
	go func() {
		log.Infof("notification worker observability server started: addr=%s", server.Addr)
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
		log.Info("notification worker shutting down")
	case err := <-consumerErrors:
		if err != nil {
			runErr = fmt.Errorf("run notification worker: %w", err)
		}
	case err := <-serverErrors:
		if err != nil {
			runErr = fmt.Errorf("serve notification worker observability HTTP: %w", err)
		}
	}

	cancelRun()
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	if err := server.Shutdown(shutdownCtx); err != nil && runErr == nil {
		runErr = fmt.Errorf("shutdown notification worker observability HTTP: %w", err)
	}

	log.Info("notification worker stopped")
	return runErr
}

type policyInstaller interface {
	Install(ctx context.Context) error
}

func installPolicy(ctx context.Context, installer policyInstaller, timeout time.Duration, log logger.Logger) error {
	installCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var lastError error
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		if err := installer.Install(installCtx); err == nil {
			log.Info("RabbitMQ delayed retry and DLQ policy installed")
			return nil
		} else {
			lastError = err
			log.Warnf("RabbitMQ policy is not ready yet: %v", err)
		}

		select {
		case <-installCtx.Done():
			return fmt.Errorf("install RabbitMQ jobs policy: %w: last error: %v", installCtx.Err(), lastError)
		case <-ticker.C:
		}
	}
}
