package httpdelivery

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aralary/edgeguard/internal/analytics/domain"
	"github.com/aralary/edgeguard/internal/analytics/usecase"
	"github.com/labstack/echo/v5"
)

const (
	defaultRange = 24 * time.Hour
	defaultLimit = 100
)

func (h *Handler) summary(c *echo.Context) error {
	filter, err := parseFilter(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
	}

	summary, err := h.usecase.GetRouteStatsSummary(c.Request().Context(), filter)
	if err != nil {
		return h.handleError(c, "get analytics summary", err)
	}

	return c.JSON(http.StatusOK, summaryResponse{
		ProjectID:          filter.ProjectID,
		From:               filter.From.Format(time.RFC3339),
		To:                 filter.To.Format(time.RFC3339),
		RouteName:          filter.RouteName,
		Method:             filter.Method,
		RequestCount:       summary.RequestCount,
		Status1xxCount:     summary.Status1xxCount,
		Status2xxCount:     summary.Status2xxCount,
		Status3xxCount:     summary.Status3xxCount,
		Status4xxCount:     summary.Status4xxCount,
		Status5xxCount:     summary.Status5xxCount,
		RateLimitedCount:   summary.RateLimitedCount,
		AverageDurationMS:  summary.AverageDurationMS,
		MaxDurationMS:      summary.MaxDurationMS,
		TotalResponseBytes: summary.TotalResponseBytes,
	})
}

func (h *Handler) hourly(c *echo.Context) error {
	filter, err := parseFilter(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
	}

	limit, err := parseLimit(c.QueryParam("limit"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
	}

	items, err := h.usecase.ListRouteStatsHourly(c.Request().Context(), filter, limit)
	if err != nil {
		return h.handleError(c, "list hourly analytics", err)
	}

	responseItems := make([]hourlyItemResponse, 0, len(items))
	for _, item := range items {
		responseItems = append(responseItems, hourlyItemResponse{
			BucketStart:        item.BucketStart,
			RouteName:          item.RouteName,
			RoutePathPrefix:    item.RoutePathPrefix,
			Method:             item.Method,
			RequestCount:       item.RequestCount,
			Status1xxCount:     item.Status1xxCount,
			Status2xxCount:     item.Status2xxCount,
			Status3xxCount:     item.Status3xxCount,
			Status4xxCount:     item.Status4xxCount,
			Status5xxCount:     item.Status5xxCount,
			RateLimitedCount:   item.RateLimitedCount,
			AverageDurationMS:  item.AverageDurationMS,
			MaxDurationMS:      item.MaxDurationMS,
			TotalResponseBytes: item.TotalResponseBytes,
		})
	}

	return c.JSON(http.StatusOK, hourlyResponse{
		ProjectID: filter.ProjectID,
		From:      filter.From.Format(time.RFC3339),
		To:        filter.To.Format(time.RFC3339),
		RouteName: filter.RouteName,
		Method:    filter.Method,
		Items:     responseItems,
	})
}

func (h *Handler) handleError(c *echo.Context, operation string, err error) error {
	if errors.Is(err, usecase.ErrInvalidProjectID) ||
		errors.Is(err, usecase.ErrInvalidTimeRange) ||
		errors.Is(err, usecase.ErrAnalyticsRangeTooLarge) ||
		errors.Is(err, usecase.ErrInvalidLimit) {
		return c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
	}

	h.log.Errorf("%s: %v", operation, err)
	return c.JSON(http.StatusInternalServerError, errorResponse{Error: "internal error"})
}

func parseFilter(c *echo.Context) (domain.RouteStatsFilter, error) {
	now := time.Now().UTC()
	to, err := parseTime(c.QueryParam("to"), now)
	if err != nil {
		return domain.RouteStatsFilter{}, err
	}
	from, err := parseTime(c.QueryParam("from"), to.Add(-defaultRange))
	if err != nil {
		return domain.RouteStatsFilter{}, err
	}

	return domain.RouteStatsFilter{
		ProjectID: strings.TrimSpace(c.Param("project_id")),
		From:      from,
		To:        to,
		RouteName: strings.TrimSpace(c.QueryParam("route_name")),
		Method:    strings.ToUpper(strings.TrimSpace(c.QueryParam("method"))),
	}, nil
}

func parseTime(value string, fallback time.Time) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback, nil
	}

	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, errors.New("from and to must use RFC3339 format")
	}
	return parsed.UTC(), nil
}

func parseLimit(value string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return defaultLimit, nil
	}

	limit, err := strconv.Atoi(value)
	if err != nil || limit <= 0 || limit > 1000 {
		return 0, usecase.ErrInvalidLimit
	}
	return limit, nil
}
