package httpdelivery

import (
	"context"

	"github.com/aralary/edgeguard/internal/analytics/domain"
	"github.com/aralary/edgeguard/internal/platform/logger"
)

type AnalyticsUsecase interface {
	GetRouteStatsSummary(context.Context, domain.RouteStatsFilter) (domain.RouteStatsSummary, error)
	ListRouteStatsHourly(context.Context, domain.RouteStatsFilter, int) ([]domain.RouteStatsHourly, error)
}

type Handler struct {
	usecase AnalyticsUsecase
	log     logger.Logger
}

func NewHandler(usecase AnalyticsUsecase, log logger.Logger) *Handler {
	return &Handler{usecase: usecase, log: log}
}
