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
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

func Run() error {
	log := logger.New()

	repo := memory.NewOrderRepository()

	listOrders := usecase.NewListOrdersUseCase(repo)
	getOrder := usecase.NewGetOrderUseCase(repo)
	createOrder := usecase.NewCreateOrderUseCase(repo)

	e := echo.New()
	e.Use(middleware.Recover())

	handler := httpdelivery.NewHandler(listOrders, getOrder, createOrder, log)
	handler.RegisterRoutes(e)

	server := &http.Server{
		Addr:              ":8081",
		Handler:           e,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Infof("demo backend started: addr=%s", server.Addr)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Errorf("demo backend failed: %v", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Info("demo backend shutting down")

	return server.Shutdown(shutdownCtx)
}
