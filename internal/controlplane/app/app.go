package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpdelivery "github.com/aralary/edgeguard/internal/controlplane/delivery/http/v1"
	controlpostgres "github.com/aralary/edgeguard/internal/controlplane/infrastructure/postgres"
	"github.com/aralary/edgeguard/internal/controlplane/usecase"
	jobsconfig "github.com/aralary/edgeguard/internal/jobs/config"
	jobshttp "github.com/aralary/edgeguard/internal/jobs/delivery/http/v1"
	jobid "github.com/aralary/edgeguard/internal/jobs/infrastructure/id"
	jobspostgres "github.com/aralary/edgeguard/internal/jobs/infrastructure/postgres"
	jobsrabbitmq "github.com/aralary/edgeguard/internal/jobs/infrastructure/rabbitmq"
	jobsusecase "github.com/aralary/edgeguard/internal/jobs/usecase"
	"github.com/aralary/edgeguard/internal/platform/logger"
	"github.com/aralary/edgeguard/internal/platform/observability"
	platformpostgres "github.com/aralary/edgeguard/internal/platform/postgres"
	platformtracing "github.com/aralary/edgeguard/internal/platform/tracing"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

const defaultHTTPAddr = ":8082"

func Run() error {
	log := logger.New()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	traceProvider, err := platformtracing.Init(ctx, "control-plane")
	if err != nil {
		return fmt.Errorf("initialize control plane tracing: %w", err)
	}
	defer shutdownTracing(traceProvider, log)

	poolCtx, cancelPool := context.WithTimeout(ctx, 10*time.Second)
	pool, err := platformpostgres.NewPool(poolCtx, platformpostgres.NewConfigFromEnv())
	cancelPool()
	if err != nil {
		return fmt.Errorf("connect to postgres: %w", err)
	}
	defer pool.Close()

	repository := controlpostgres.New(pool)
	controlPlaneUsecase := usecase.New(repository, repository, repository)

	jobsRuntimeConfig, err := jobsconfig.Load()
	if err != nil {
		return fmt.Errorf("load jobs config: %w", err)
	}

	jobPublisher, err := jobsrabbitmq.New(jobsrabbitmq.Config{
		URL:            jobsRuntimeConfig.RabbitMQURL,
		Exchange:       jobsRuntimeConfig.Exchange,
		Queue:          jobsRuntimeConfig.Queue,
		PublishTimeout: jobsRuntimeConfig.PublishTimeout,
	})
	if err != nil {
		return fmt.Errorf("create jobs publisher: %w", err)
	}
	defer jobPublisher.Close()

	jobsRepository := jobspostgres.New(pool)
	jobsUsecase := jobsusecase.New(jobsusecase.Dependencies{
		Publisher:   jobPublisher,
		Repository:  jobsRepository,
		IDGenerator: jobid.NewGenerator(),
	})

	metrics := observability.NewMetrics("control-plane")
	readiness := observability.NewReadiness("control-plane",
		observability.Check{Name: "postgres", Run: pool.Ping},
		observability.Check{Name: "rabbitmq", Run: jobPublisher.Ping},
	)

	e := echo.New()
	e.Use(middleware.Recover())
	e.Use(metrics.Middleware())
	e.Use(platformtracing.RouteMiddleware())
	observability.Register(e, metrics, readiness)

	handler := httpdelivery.NewHandler(controlPlaneUsecase, log)
	handler.RegisterRoutes(e)

	jobsHandler := jobshttp.NewHandler(jobsUsecase, log)
	jobsHandler.RegisterRoutes(e)

	server := &http.Server{
		Addr:              httpAddr(),
		Handler:           platformtracing.WrapHTTPHandler("control-plane", e),
		ReadHeaderTimeout: 5 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		log.Infof("control plane started: addr=%s", server.Addr)

		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			serverErrors <- err
			return
		}

		serverErrors <- nil
	}()

	select {
	case <-ctx.Done():
		log.Info("control plane shutting down")
	case err := <-serverErrors:
		if err != nil {
			return fmt.Errorf("serve control plane http: %w", err)
		}

		return nil
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown control plane http server: %w", err)
	}

	return nil
}

func httpAddr() string {
	addr := os.Getenv("CONTROL_PLANE_HTTP_ADDR")
	if addr == "" {
		return defaultHTTPAddr
	}

	return addr
}

func shutdownTracing(provider *platformtracing.Provider, log logger.Logger) {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := provider.Shutdown(shutdownCtx); err != nil {
		log.Warnf("shutdown OpenTelemetry tracing: %v", err)
	}
}
