// internal/config/tailscale_test.go
package config

import "testing"

var tailscaleFixture = `{
  "Self": {
    "DNSName": "homepc.tail-4a9c2e.ts.net.",
    "HostName": "homepc"
  }
}`

func TestParseTailscaleStatus(t *testing.T) {
	got := parseTailscaleStatus([]byte(tailscaleFixture))
	if got != "homepc.tail-4a9c2e.ts.net" {
		t.Fatalf("want trimmed DNSName, got %q", got)
	}
}

func TestParseTailscaleStatus_Empty(t *testing.T) {
	if got := parseTailscaleStatus([]byte(`{}`)); got != "" {
		t.Fatalf("want empty, got %q", got)
	}
}

func TestParseTailscaleStatus_Invalid(t *testing.T) {
	if got := parseTailscaleStatus([]byte(`not json`)); got != "" {
		t.Fatalf("want empty, got %q", got)
	}
}
