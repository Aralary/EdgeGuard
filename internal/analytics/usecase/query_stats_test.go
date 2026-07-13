package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aralary/edgeguard/internal/analytics/domain"
)

type gatewayStatsRepositoryStub struct {
	summary domain.RouteStatsSummary
	items   []domain.RouteStatsHourly
	filter  domain.RouteStatsFilter
	limit   int
	err     error
}

func (s *gatewayStatsRepositoryStub) GetRouteStatsSummary(
	_ context.Context,
	filter domain.RouteStatsFilter,
) (domain.RouteStatsSummary, error) {
	s.filter = filter
	return s.summary, s.err
}

func (s *gatewayStatsRepositoryStub) ListRouteStatsHourly(
	_ context.Context,
	filter domain.RouteStatsFilter,
	limit int,
) ([]domain.RouteStatsHourly, error) {
	s.filter = filter
	s.limit = limit
	return s.items, s.err
}

func TestGetRouteStatsSummaryNormalizesFilter(t *testing.T) {
	repository := &gatewayStatsRepositoryStub{summary: domain.RouteStatsSummary{RequestCount: 3}}
	uc := New(Dependencies{GatewayStatsRepository: repository})
	from := time.Date(2026, 7, 13, 10, 0, 0, 0, time.FixedZone("test", 2*60*60))
	to := from.Add(time.Hour)

	result, err := uc.GetRouteStatsSummary(context.Background(), domain.RouteStatsFilter{
		ProjectID: "11111111-1111-1111-1111-111111111111",
		From:      from,
		To:        to,
		RouteName: " orders ",
		Method:    " get ",
	})
	if err != nil {
		t.Fatalf("GetRouteStatsSummary() error = %v", err)
	}
	if result.RequestCount != 3 {
		t.Fatalf("RequestCount = %d, want 3", result.RequestCount)
	}
	if repository.filter.RouteName != "orders" || repository.filter.Method != "GET" {
		t.Fatalf("unexpected normalized filter: %#v", repository.filter)
	}
	if repository.filter.From.Location() != time.UTC || repository.filter.To.Location() != time.UTC {
		t.Fatalf("filter times must be UTC: %#v", repository.filter)
	}
}

func TestGetRouteStatsSummaryRejectsInvalidProjectID(t *testing.T) {
	uc := New(Dependencies{GatewayStatsRepository: &gatewayStatsRepositoryStub{}})
	_, err := uc.GetRouteStatsSummary(context.Background(), domain.RouteStatsFilter{
		ProjectID: "not-a-uuid",
		From:      time.Now().Add(-time.Hour),
		To:        time.Now(),
	})
	if !errors.Is(err, ErrInvalidProjectID) {
		t.Fatalf("error = %v, want ErrInvalidProjectID", err)
	}
}

func TestGetRouteStatsSummaryRejectsLargeRange(t *testing.T) {
	uc := New(Dependencies{GatewayStatsRepository: &gatewayStatsRepositoryStub{}})
	to := time.Now().UTC()
	_, err := uc.GetRouteStatsSummary(context.Background(), domain.RouteStatsFilter{
		ProjectID: "11111111-1111-1111-1111-111111111111",
		From:      to.Add(-91 * 24 * time.Hour),
		To:        to,
	})
	if !errors.Is(err, ErrAnalyticsRangeTooLarge) {
		t.Fatalf("error = %v, want ErrAnalyticsRangeTooLarge", err)
	}
}

func TestListRouteStatsHourlyRejectsInvalidLimit(t *testing.T) {
	uc := New(Dependencies{GatewayStatsRepository: &gatewayStatsRepositoryStub{}})
	to := time.Now().UTC()
	_, err := uc.ListRouteStatsHourly(context.Background(), domain.RouteStatsFilter{
		ProjectID: "11111111-1111-1111-1111-111111111111",
		From:      to.Add(-time.Hour),
		To:        to,
	}, 1001)
	if !errors.Is(err, ErrInvalidLimit) {
		t.Fatalf("error = %v, want ErrInvalidLimit", err)
	}
}
