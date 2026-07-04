package app

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpdelivery "github.com/aralary/edgeguard/internal/gateway/delivery/http/v1"
	"github.com/aralary/edgeguard/internal/gateway/infrastructure/config"
	"github.com/aralary/edgeguard/internal/gateway/infrastructure/proxy"
	"github.com/aralary/edgeguard/internal/gateway/usecase"
)

func Run() error {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	configPath := os.Getenv("GATEWAY_CONFIG_PATH")
	if configPath == "" {
		configPath = "configs/gateway.yaml"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	routeRepo := config.NewYAMLRouteRepository(cfg.DomainRoutes())
	resolveRoute := usecase.NewResolveRouteUseCase(routeRepo)
	upstreamProxy := proxy.NewHTTPUtilProxy()

	handler := httpdelivery.NewHandler(resolveRoute, upstreamProxy, log)

	var httpHandler http.Handler = handler
	httpHandler = httpdelivery.RequestID(httpHandler)
	httpHandler = httpdelivery.Logging(log, httpHandler)

	server := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           httpHandler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Info("gateway started", slog.String("addr", server.Addr))

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("gateway failed", slog.String("error", err.Error()))
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