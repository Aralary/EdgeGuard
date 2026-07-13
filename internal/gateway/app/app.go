package app

import (
	"context"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	gatewayconfig "github.com/aralary/edgeguard/internal/gateway/config"
	httpdelivery "github.com/aralary/edgeguard/internal/gateway/delivery/http/v1"
	authclient "github.com/aralary/edgeguard/internal/gateway/infrastructure/auth"
	yamlconfig "github.com/aralary/edgeguard/internal/gateway/infrastructure/config"
	"github.com/aralary/edgeguard/internal/gateway/infrastructure/controlplane"
	kafkaproducer "github.com/aralary/edgeguard/internal/gateway/infrastructure/kafka"
	"github.com/aralary/edgeguard/internal/gateway/infrastructure/memory"
	"github.com/aralary/edgeguard/internal/gateway/infrastructure/proxy"
	"github.com/aralary/edgeguard/internal/gateway/infrastructure/ratelimit"
	"github.com/aralary/edgeguard/internal/gateway/usecase"
	"github.com/aralary/edgeguard/internal/gateway/worker"
	"github.com/aralary/edgeguard/internal/platform/logger"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

func Run() error {
	log := logger.New()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	runtimeConfig, err := gatewayconfig.Load()
	if err != nil {
		return err
	}

	gatewayYAMLConfig, err := yamlconfig.Load(runtimeConfig.ConfigPath)
	if err != nil {
		return err
	}

	var routeSource usecase.RouteSource
	if runtimeConfig.ControlPlaneURL != "" {
		routeSource, err = controlplane.New(runtimeConfig.ControlPlaneURL, nil)
		if err != nil {
			return err
		}
	}

	var apiKeyValidator usecase.APIKeyValidator
	if runtimeConfig.AuthServiceURL != "" {
		apiKeyValidator, err = authclient.New(runtimeConfig.AuthServiceURL, nil)
		if err != nil {
			return err
		}
	}

	var rateLimiter *ratelimit.Limiter
	if runtimeConfig.RedisURL != "" {
		rateLimiter, err = ratelimit.New(
			runtimeConfig.RedisURL,
			runtimeConfig.RedisRateLimitPrefix,
		)
		if err != nil {
			return err
		}
		defer rateLimiter.Close()
	}

	var accessEventPublisher usecase.AccessEventPublisher
	if len(runtimeConfig.KafkaBrokers) > 0 {
		producer, producerErr := kafkaproducer.New(kafkaproducer.Config{
			Brokers:  runtimeConfig.KafkaBrokers,
			Topic:    runtimeConfig.KafkaAccessTopic,
			ClientID: runtimeConfig.KafkaClientID,
		}, log)
		if producerErr != nil {
			return producerErr
		}

		accessEventPublisher = producer
		defer func() {
			if closeErr := producer.Close(); closeErr != nil {
				log.Warnf("close gateway kafka producer: %v", closeErr)
			}
		}()
	}

	routeRepository := memory.NewRouteRepository(gatewayYAMLConfig.DomainRoutes())
	gatewayUsecase := usecase.New(usecase.Dependencies{
		RouteRepository:      routeRepository,
		RouteSource:          routeSource,
		APIKeyValidator:      apiKeyValidator,
		RateLimiter:          rateLimiter,
		AccessEventPublisher: accessEventPublisher,
	})

	if routeSource != nil {
		count, refreshErr := gatewayUsecase.RefreshRoutes(ctx)
		if refreshErr != nil {
			log.Warnf("initial gateway routes refresh failed, using configured fallback routes: %v", refreshErr)
		} else {
			log.Infof("initial gateway routes loaded: routes=%d", count)
		}

		go worker.RunRoutesRefresh(
			ctx,
			runtimeConfig.RoutesRefreshInterval,
			gatewayUsecase,
			log,
		)
	}

	upstreamProxy := proxy.NewHTTPUtilProxy()

	e := echo.New()

	e.Use(middleware.Recover())
	e.Use(httpdelivery.RequestID())
	e.Use(httpdelivery.AccessEvents(gatewayUsecase, log))
	e.Use(httpdelivery.Logging(log))

	handler := httpdelivery.NewHandler(gatewayUsecase, upstreamProxy, log)
	handler.RegisterRoutes(e)

	server := &http.Server{
		Addr:              gatewayYAMLConfig.HTTP.Addr,
		Handler:           e,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Infof("gateway started: addr=%s", server.Addr)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Errorf("gateway failed: %v", err)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Info("gateway shutting down")

	return server.Shutdown(shutdownCtx)
}
