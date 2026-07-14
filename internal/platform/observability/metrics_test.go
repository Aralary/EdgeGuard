package observability

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
)

func TestMetricsMiddlewareRecordsRequest(t *testing.T) {
	metrics := NewMetrics("test")
	e := echo.New()
	e.Use(metrics.Middleware())
	e.GET("/items/:id", func(c *echo.Context) error {
		return c.String(http.StatusCreated, "created")
	})

	recorder := httptest.NewRecorder()
	e.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/items/42", nil))
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusCreated)
	}

	families, err := metrics.Gatherer().Gather()
	if err != nil {
		t.Fatalf("gather metrics: %v", err)
	}

	for _, family := range families {
		if family.GetName() == "edgeguard_http_requests_total" {
			if len(family.Metric) != 1 || family.Metric[0].Counter.GetValue() != 1 {
				t.Fatalf("unexpected request counter: %+v", family.Metric)
			}
			return
		}
	}

	t.Fatal("edgeguard_http_requests_total metric not found")
}
