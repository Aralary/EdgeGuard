package webhook

import (
	"net"
	"testing"
)

func TestBlockedIP(t *testing.T) {
	testCases := []string{"127.0.0.1", "10.0.0.1", "192.168.1.2", "169.254.169.254", "::1", "fc00::1"}
	for _, rawIP := range testCases {
		if !blockedIP(net.ParseIP(rawIP)) {
			t.Fatalf("blockedIP(%q) = false, want true", rawIP)
		}
	}
}

func TestPublicIPIsAllowed(t *testing.T) {
	if blockedIP(net.ParseIP("8.8.8.8")) {
		t.Fatal("blockedIP(public IP) = true, want false")
	}
}
