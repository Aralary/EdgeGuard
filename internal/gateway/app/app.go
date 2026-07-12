package app

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpdelivery "github.com/aralary/edgeguard/internal/gateway/delivery/http/v1"
	"github.com/aralary/edgeguard/internal/gateway/infrastructure/config"
	"github.com/aralary/edgeguard/internal/gateway/infrastructure/memory"
	"github.com/aralary/edgeguard/internal/gateway/infrastructure/proxy"
	"github.com/aralary/edgeguard/internal/gateway/usecase"
	"github.com/aralary/edgeguard/internal/platform/logger"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

func Run() error {
	log := logger.New()

	configPath := os.Getenv("GATEWAY_CONFIG_PATH")
	if configPath == "" {
		configPath = "configs/gateway.yaml"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	routeRepo := memory.NewRouteRepository(cfg.DomainRoutes())
	gatewayUsecase := usecase.New(routeRepo)
	upstreamProxy := proxy.NewHTTPUtilProxy()

	e := echo.New()

	e.Use(middleware.Recover())
	e.Use(httpdelivery.RequestID())
	e.Use(httpdelivery.Logging(log))

	handler := httpdelivery.NewHandler(gatewayUsecase, upstreamProxy, log)
	handler.RegisterRoutes(e)

	server := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           e,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Infof("gateway started: addr=%s", server.Addr)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Errorf("gateway failed: %v", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Info("gateway shutting down")

	return server.Shutdown(shutdownCtx)
}
