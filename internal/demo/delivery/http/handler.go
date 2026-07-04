package httpdelivery

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/aralary/edgeguard/internal/demo/domain"
	"github.com/aralary/edgeguard/internal/demo/infrastructure/memory"
	"github.com/aralary/edgeguard/internal/demo/usecase"
)

type Handler struct {
	listOrders   *usecase.ListOrdersUseCase
	getOrder     *usecase.GetOrderUseCase
	createOrder  *usecase.CreateOrderUseCase
	log          *slog.Logger
}

func NewHandler(
	listOrders *usecase.ListOrdersUseCase,
	getOrder *usecase.GetOrderUseCase,
	createOrder *usecase.CreateOrderUseCase,
	log *slog.Logger,
) *Handler {
	return &Handler{
		listOrders:  listOrders,
		getOrder:    getOrder,
		createOrder: createOrder,
		log:         log,
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("GET /orders", h.list)
	mux.HandleFunc("GET /orders/{id}", h.get)
	mux.HandleFunc("POST /orders", h.create)
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	orders, err := h.listOrders.Execute(r.Context())
	if err != nil {
		h.log.Error("failed to list orders", slog.String("error", err.Error()))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, orders)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	order, err := h.getOrder.Execute(r.Context(), id)
	if err != nil {
		if errors.Is(err, memory.ErrOrderNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "order not found"})
			return
		}

		h.log.Error("failed to get order", slog.String("error", err.Error()))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, order)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req domain.Order

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	order, err := h.createOrder.Execute(r.Context(), req)
	if err != nil {
		h.log.Error("failed to create order", slog.String("error", err.Error()))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusCreated, order)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(value); err != nil {
		slog.Error("failed to encode response", slog.String("error", err.Error()))
	}
}