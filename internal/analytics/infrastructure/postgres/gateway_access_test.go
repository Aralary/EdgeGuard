package postgres

import (
	"testing"

	"github.com/aralary/edgeguard/internal/platform/events"
)

func TestStatusClassCounts(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		want       statusCounts
	}{
		{name: "informational", statusCode: 101, want: statusCounts{status1xx: 1}},
		{name: "success", statusCode: 200, want: statusCounts{status2xx: 1}},
		{name: "redirect", statusCode: 302, want: statusCounts{status3xx: 1}},
		{name: "client error", statusCode: 429, want: statusCounts{status4xx: 1}},
		{name: "server error", statusCode: 503, want: statusCounts{status5xx: 1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := statusClassCounts(tt.statusCode); got != tt.want {
				t.Fatalf("statusClassCounts(%d) = %#v, want %#v", tt.statusCode, got, tt.want)
			}
		})
	}
}

func TestShouldAggregate(t *testing.T) {
	valid := events.GatewayAccessEvent{
		ProjectID:       "11111111-1111-1111-1111-111111111111",
		RouteName:       "orders",
		RoutePathPrefix: "/api",
	}
	if !shouldAggregate(valid) {
		t.Fatal("shouldAggregate(valid) = false, want true")
	}

	valid.ProjectID = ""
	if shouldAggregate(valid) {
		t.Fatal("shouldAggregate(without project) = true, want false")
	}
}
