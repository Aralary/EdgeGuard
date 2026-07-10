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
	"github.com/aralary/edgeguard/internal/platform/logger"
	platformpostgres "github.com/aralary/edgeguard/internal/platform/postgres"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

const defaultHTTPAddr = ":8082"

func Run() error {
	log := logger.New()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	poolCtx, cancelPool := context.WithTimeout(ctx, 10*time.Second)
	pool, err := platformpostgres.NewPool(poolCtx, platformpostgres.NewConfigFromEnv())
	cancelPool()
	if err != nil {
		return fmt.Errorf("connect to postgres: %w", err)
	}
	defer pool.Close()

	repository := controlpostgres.New(pool)
	controlPlaneUsecase := usecase.New(repository, repository, repository)

	e := echo.New()
	e.Use(middleware.Recover())

	handler := httpdelivery.NewHandler(controlPlaneUsecase, log)
	handler.RegisterRoutes(e)

	server := &http.Server{
		Addr:              httpAddr(),
		Handler:           e,
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
