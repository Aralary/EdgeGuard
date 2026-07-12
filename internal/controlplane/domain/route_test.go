package domain

import "testing"

func TestNewRouteKeepsAuthPolicy(t *testing.T) {
	route, err := NewRoute(
		"service-1",
		"protected-orders",
		"/api/v1",
		true,
		0,
		true,
		true,
	)
	if err != nil {
		t.Fatalf("NewRoute() error = %v", err)
	}

	if !route.AuthRequired {
		t.Fatal("AuthRequired = false, want true")
	}
	if route.TimeoutMS != DefaultRouteTimeoutMS {
		t.Fatalf("TimeoutMS = %d, want %d", route.TimeoutMS, DefaultRouteTimeoutMS)
	}
}
