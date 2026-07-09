package httpdelivery

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/aralary/edgeguard/internal/demo/domain"
	"github.com/aralary/edgeguard/internal/demo/infrastructure/memory"
	"github.com/labstack/echo/v5"
)

func (h *Handler) health(c *echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func (h *Handler) list(c *echo.Context) error {
	orders, err := h.listOrders.Execute(c.Request().Context())
	if err != nil {
		h.log.Errorf("failed to list orders: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	return c.JSON(http.StatusOK, orders)
}

func (h *Handler) get(c *echo.Context) error {
	id := c.Param("id")

	order, err := h.getOrder.Execute(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, memory.ErrOrderNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "order not found"})
		}

		h.log.Errorf("failed to get order: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	return c.JSON(http.StatusOK, order)
}

func (h *Handler) create(c *echo.Context) error {
	var req domain.Order

	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	order, err := h.createOrder.Execute(c.Request().Context(), req)
	if err != nil {
		h.log.Errorf("failed to create order: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	return c.JSON(http.StatusCreated, order)
}
