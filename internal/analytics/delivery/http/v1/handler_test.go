package httpdelivery

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aralary/edgeguard/internal/analytics/domain"
	"github.com/labstack/echo/v5"
)

type analyticsUsecaseStub struct {
	summaryFn func(context.Context, domain.RouteStatsFilter) (domain.RouteStatsSummary, error)
	hourlyFn  func(context.Context, domain.RouteStatsFilter, int) ([]domain.RouteStatsHourly, error)
}

func (s analyticsUsecaseStub) GetRouteStatsSummary(
	ctx context.Context,
	filter domain.RouteStatsFilter,
) (domain.RouteStatsSummary, error) {
	return s.summaryFn(ctx, filter)
}

func (s analyticsUsecaseStub) ListRouteStatsHourly(
	ctx context.Context,
	filter domain.RouteStatsFilter,
	limit int,
) ([]domain.RouteStatsHourly, error) {
	return s.hourlyFn(ctx, filter, limit)
}

type testLogger struct{}

func (testLogger) Debug(args ...interface{})                 {}
func (testLogger) Debugf(format string, args ...interface{}) {}
func (testLogger) Info(args ...interface{})                  {}
func (testLogger) Infof(format string, args ...interface{})  {}
func (testLogger) Warn(args ...interface{})                  {}
func (testLogger) Warnf(format string, args ...interface{})  {}
func (testLogger) Error(args ...interface{})                 {}
func (testLogger) Errorf(format string, args ...interface{}) {}

func TestSummary(t *testing.T) {
	stub := analyticsUsecaseStub{
		summaryFn: func(_ context.Context, filter domain.RouteStatsFilter) (domain.RouteStatsSummary, error) {
			if filter.RouteName != "orders" || filter.Method != "GET" {
				t.Fatalf("unexpected filter: %#v", filter)
			}
			return domain.RouteStatsSummary{RequestCount: 3, Status2xxCount: 3, AverageDurationMS: 12.5}, nil
		},
		hourlyFn: func(context.Context, domain.RouteStatsFilter, int) ([]domain.RouteStatsHourly, error) {
			return nil, nil
		},
	}

	recorder := performRequest(t, stub, http.MethodGet,
		"/api/v1/projects/11111111-1111-1111-1111-111111111111/analytics/summary?route_name=orders&method=get")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", recorder.Code, recorder.Body.String())
	}

	var response summaryResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.RequestCount != 3 || response.Status2xxCount != 3 || response.AverageDurationMS != 12.5 {
		t.Fatalf("unexpected response: %#v", response)
	}
}

func TestHourlyRejectsInvalidLimit(t *testing.T) {
	stub := analyticsUsecaseStub{
		summaryFn: func(context.Context, domain.RouteStatsFilter) (domain.RouteStatsSummary, error) {
			return domain.RouteStatsSummary{}, nil
		},
		hourlyFn: func(context.Context, domain.RouteStatsFilter, int) ([]domain.RouteStatsHourly, error) {
			t.Fatal("hourly usecase must not be called")
			return nil, nil
		},
	}

	recorder := performRequest(t, stub, http.MethodGet,
		"/api/v1/projects/11111111-1111-1111-1111-111111111111/analytics/hourly?limit=1001")
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
}

func TestHourlyReturnsEmptyArray(t *testing.T) {
	stub := analyticsUsecaseStub{
		summaryFn: func(context.Context, domain.RouteStatsFilter) (domain.RouteStatsSummary, error) {
			return domain.RouteStatsSummary{}, nil
		},
		hourlyFn: func(_ context.Context, _ domain.RouteStatsFilter, limit int) ([]domain.RouteStatsHourly, error) {
			if limit != defaultLimit {
				t.Fatalf("limit = %d, want %d", limit, defaultLimit)
			}
			return []domain.RouteStatsHourly{}, nil
		},
	}

	recorder := performRequest(t, stub, http.MethodGet,
		"/api/v1/projects/11111111-1111-1111-1111-111111111111/analytics/hourly")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	var response hourlyResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Items == nil || len(response.Items) != 0 {
		t.Fatalf("items = %#v, want empty array", response.Items)
	}
}

func performRequest(t *testing.T, stub analyticsUsecaseStub, method string, path string) *httptest.ResponseRecorder {
	t.Helper()
	e := echo.New()
	NewHandler(stub, testLogger{}).RegisterRoutes(e)
	req := httptest.NewRequest(method, path, nil)
	recorder := httptest.NewRecorder()
	e.ServeHTTP(recorder, req)
	return recorder
}
