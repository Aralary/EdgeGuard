package app

import (
	"context"
	"fmt"
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
	platformpostgres "github.com/aralary/edgeguard/internal/platform/postgres"
)

func Run() error {
	log := logger.New()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	config, err := jobworkerconfig.Load()
	if err != nil {
		return fmt.Errorf("load notification worker config: %w", err)
	}

	poolCtx, cancelPool := context.WithTimeout(ctx, 10*time.Second)
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
	if err := installPolicy(ctx, policyInstaller, config.PolicyInstallTimeout, log); err != nil {
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

	log.Infof(
		"notification worker started: queue=%s dead_queue=%s prefetch=%d retry=%s..%s",
		config.Queue,
		config.DeadQueue,
		config.Prefetch,
		config.RetryMin,
		config.RetryMax,
	)

	if err := consumer.Run(ctx, jobUsecase, jobStatusRepository, log); err != nil {
		return fmt.Errorf("run notification worker: %w", err)
	}

	log.Info("notification worker stopped")
	return nil
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
