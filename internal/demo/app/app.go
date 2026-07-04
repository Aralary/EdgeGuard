package app

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpdelivery "github.com/aralary/edgeguard/internal/demo/delivery/http"
	"github.com/aralary/edgeguard/internal/demo/infrastructure/memory"
	"github.com/aralary/edgeguard/internal/demo/usecase"
)

func Run() error {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	repo := memory.NewOrderRepository()

	listOrders := usecase.NewListOrdersUseCase(repo)
	getOrder := usecase.NewGetOrderUseCase(repo)
	createOrder := usecase.NewCreateOrderUseCase(repo)

	handler := httpdelivery.NewHandler(listOrders, getOrder, createOrder, log)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	server := &http.Server{
		Addr:              ":8081",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Info("demo backend started", slog.String("addr", server.Addr))

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("demo backend failed", slog.String("error", err.Error()))
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