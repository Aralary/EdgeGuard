package httpdelivery

import (
	"net/http"

	"github.com/aralary/edgeguard/internal/jobs/usecase"
	"github.com/labstack/echo/v5"
)

func (h *Handler) submitWebhook(c *echo.Context) error {
	var request submitWebhookRequest
	if err := decodeRequest(c, &request); err != nil {
		return h.handleError(c, "decode webhook job request", err)
	}

	receipt, err := h.usecase.SubmitWebhook(c.Request().Context(), usecase.SubmitWebhookInput{
		URL:         request.URL,
		Method:      request.Method,
		Headers:     request.Headers,
		Body:        request.Body,
		MaxAttempts: request.MaxAttempts,
	})
	if err != nil {
		return h.handleError(c, "submit webhook job", err)
	}

	return c.JSON(http.StatusAccepted, newJobReceiptResponse(receipt))
}

func (h *Handler) submitReport(c *echo.Context) error {
	var request submitReportRequest
	if err := decodeRequest(c, &request); err != nil {
		return h.handleError(c, "decode report job request", err)
	}

	receipt, err := h.usecase.SubmitReport(c.Request().Context(), usecase.SubmitReportInput{
		ProjectID:   request.ProjectID,
		From:        request.From,
		To:          request.To,
		Format:      request.Format,
		MaxAttempts: request.MaxAttempts,
	})
	if err != nil {
		return h.handleError(c, "submit report job", err)
	}

	return c.JSON(http.StatusAccepted, newJobReceiptResponse(receipt))
}

func (h *Handler) submitCleanup(c *echo.Context) error {
	var request submitCleanupRequest
	if err := decodeRequest(c, &request); err != nil {
		return h.handleError(c, "decode cleanup job request", err)
	}

	receipt, err := h.usecase.SubmitCleanup(c.Request().Context(), usecase.SubmitCleanupInput{
		Before:      request.Before,
		BatchSize:   request.BatchSize,
		MaxAttempts: request.MaxAttempts,
	})
	if err != nil {
		return h.handleError(c, "submit cleanup job", err)
	}

	return c.JSON(http.StatusAccepted, newJobReceiptResponse(receipt))
}
