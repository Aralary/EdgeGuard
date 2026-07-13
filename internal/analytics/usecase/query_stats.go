package usecase

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/aralary/edgeguard/internal/analytics/domain"
)

const maxAnalyticsRange = 90 * 24 * time.Hour

func (u *Usecase) GetRouteStatsSummary(
	ctx context.Context,
	filter domain.RouteStatsFilter,
) (domain.RouteStatsSummary, error) {
	filter, err := normalizeStatsFilter(filter)
	if err != nil {
		return domain.RouteStatsSummary{}, err
	}
	if u == nil || u.gatewayStatsRepository == nil {
		return domain.RouteStatsSummary{}, fmt.Errorf("gateway stats repository is required")
	}

	summary, err := u.gatewayStatsRepository.GetRouteStatsSummary(ctx, filter)
	if err != nil {
		return domain.RouteStatsSummary{}, fmt.Errorf("get route stats summary: %w", err)
	}

	return summary, nil
}

func (u *Usecase) ListRouteStatsHourly(
	ctx context.Context,
	filter domain.RouteStatsFilter,
	limit int,
) ([]domain.RouteStatsHourly, error) {
	filter, err := normalizeStatsFilter(filter)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 1000 {
		return nil, ErrInvalidLimit
	}
	if u == nil || u.gatewayStatsRepository == nil {
		return nil, fmt.Errorf("gateway stats repository is required")
	}

	items, err := u.gatewayStatsRepository.ListRouteStatsHourly(ctx, filter, limit)
	if err != nil {
		return nil, fmt.Errorf("list hourly route stats: %w", err)
	}
	if items == nil {
		items = []domain.RouteStatsHourly{}
	}

	return items, nil
}

func normalizeStatsFilter(filter domain.RouteStatsFilter) (domain.RouteStatsFilter, error) {
	filter.ProjectID = strings.TrimSpace(filter.ProjectID)
	if !isUUID(filter.ProjectID) {
		return domain.RouteStatsFilter{}, ErrInvalidProjectID
	}

	filter.From = filter.From.UTC()
	filter.To = filter.To.UTC()
	if filter.From.IsZero() || filter.To.IsZero() || !filter.From.Before(filter.To) {
		return domain.RouteStatsFilter{}, ErrInvalidTimeRange
	}
	if filter.To.Sub(filter.From) > maxAnalyticsRange {
		return domain.RouteStatsFilter{}, ErrAnalyticsRangeTooLarge
	}

	// Statistics are stored in hourly buckets. Expand the requested interval to
	// complete bucket boundaries so the current and boundary hours are included.
	filter.From = filter.From.Truncate(time.Hour)
	filter.To = ceilHour(filter.To)

	filter.RouteName = strings.TrimSpace(filter.RouteName)
	filter.Method = strings.ToUpper(strings.TrimSpace(filter.Method))

	return filter, nil
}

func isUUID(value string) bool {
	if len(value) != 36 || value[8] != '-' || value[13] != '-' || value[18] != '-' || value[23] != '-' {
		return false
	}

	compact := strings.ReplaceAll(value, "-", "")
	if len(compact) != 32 {
		return false
	}

	_, err := hex.DecodeString(compact)
	return err == nil
}

func ceilHour(value time.Time) time.Time {
	truncated := value.Truncate(time.Hour)
	if value.Equal(truncated) {
		return truncated
	}

	return truncated.Add(time.Hour)
}
