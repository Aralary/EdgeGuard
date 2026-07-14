package app

import (
	"context"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	httpdelivery "github.com/aralary/edgeguard/internal/demo/delivery/http/v1"
	"github.com/aralary/edgeguard/internal/demo/infrastructure/memory"
	"github.com/aralary/edgeguard/internal/demo/usecase"
	"github.com/aralary/edgeguard/internal/platform/logger"
	"github.com/aralary/edgeguard/internal/platform/observability"
	platformtracing "github.com/aralary/edgeguard/internal/platform/tracing"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

func Run() error {
	log := logger.New()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	traceProvider, err := platformtracing.Init(ctx, "demo-backend")
	if err != nil {
		return err
	}
	defer shutdownTracing(traceProvider, log)

	repo := memory.NewOrderRepository()

	demoUsecase := usecase.New(repo)

	metrics := observability.NewMetrics("demo-backend")
	readiness := observability.NewReadiness("demo-backend")

	e := echo.New()
	e.Use(middleware.Recover())
	e.Use(metrics.Middleware())
	e.Use(platformtracing.RouteMiddleware())
	observability.Register(e, metrics, readiness)

	handler := httpdelivery.NewHandler(demoUsecase, log)
	handler.RegisterRoutes(e)

	server := &http.Server{
		Addr:              ":8081",
		Handler:           platformtracing.WrapHTTPHandler("demo-backend", e),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Infof("demo backend started: addr=%s", server.Addr)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Errorf("demo backend failed: %v", err)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Info("demo backend shutting down")

	return server.Shutdown(shutdownCtx)
}

func shutdownTracing(provider *platformtracing.Provider, log logger.Logger) {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := provider.Shutdown(shutdownCtx); err != nil {
		log.Warnf("shutdown OpenTelemetry tracing: %v", err)
	}
}
