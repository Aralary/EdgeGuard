package rabbitmq

import "testing"

func TestRegexpQuote(t *testing.T) {
	got := regexpQuote("edgeguard.jobs.main.v1")
	want := `edgeguard\.jobs\.main\.v1`
	if got != want {
		t.Fatalf("regexpQuote() = %q, want %q", got, want)
	}
}
