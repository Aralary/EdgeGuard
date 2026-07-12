package app

import (
	"context"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/aralary/edgeguard/internal/auth/config"
	httpdelivery "github.com/aralary/edgeguard/internal/auth/delivery/http/v1"
	authpostgres "github.com/aralary/edgeguard/internal/auth/infrastructure/postgres"
	"github.com/aralary/edgeguard/internal/auth/infrastructure/security"
	"github.com/aralary/edgeguard/internal/auth/usecase"
	"github.com/aralary/edgeguard/internal/platform/logger"
	platformpostgres "github.com/aralary/edgeguard/internal/platform/postgres"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

func Run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load auth config: %w", err)
	}

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

	passwordHasher, err := security.NewPasswordHasher(cfg.BcryptCost)
	if err != nil {
		return fmt.Errorf("create password hasher: %w", err)
	}

	tokenService, err := security.NewTokenService(security.TokenServiceConfig{
		JWTSecret:        cfg.JWTSecret,
		JWTIssuer:        cfg.JWTIssuer,
		JWTAudience:      cfg.JWTAudience,
		AccessTokenTTL:   cfg.AccessTokenTTL,
		RefreshTokenSize: cfg.RefreshTokenSize,
	})
	if err != nil {
		return fmt.Errorf("create token service: %w", err)
	}

	repository := authpostgres.New(pool)
	authUsecase := usecase.New(
		repository,
		repository,
		passwordHasher,
		tokenService,
		nil,
		usecase.Config{RefreshTokenTTL: cfg.RefreshTokenTTL},
	)

	e := echo.New()
	e.Use(middleware.Recover())

	handler := httpdelivery.NewHandler(authUsecase, log)
	handler.RegisterRoutes(e)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           e,
		ReadHeaderTimeout: 5 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		log.Infof("auth service started: addr=%s", server.Addr)

		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			serverErrors <- err
			return
		}

		serverErrors <- nil
	}()

	select {
	case <-ctx.Done():
		log.Info("auth service shutting down")
	case err := <-serverErrors:
		if err != nil {
			return fmt.Errorf("serve auth http: %w", err)
		}

		return nil
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown auth http server: %w", err)
	}

	return nil
}
