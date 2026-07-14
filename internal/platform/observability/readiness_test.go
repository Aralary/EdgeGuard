package observability

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
)

func TestReadinessHandler(t *testing.T) {
	tests := []struct {
		name       string
		checks     []Check
		wantStatus int
	}{
		{
			name:       "ready",
			checks:     []Check{{Name: "postgres", Run: func(context.Context) error { return nil }}},
			wantStatus: http.StatusOK,
		},
		{
			name:       "not ready",
			checks:     []Check{{Name: "postgres", Run: func(context.Context) error { return errors.New("unavailable") }}},
			wantStatus: http.StatusServiceUnavailable,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			e := echo.New()
			readiness := NewReadiness("test", test.checks...)
			e.GET("/ready", readiness.Handler)

			recorder := httptest.NewRecorder()
			e.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/ready", nil))
			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, test.wantStatus)
			}
		})
	}
}
