// internal/config/tailscale.go
package config

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"
	"time"
)

// autoDetectHost shells out to `tailscale status --json` and returns the
// MagicDNS name of Self, or "" on any failure.
func autoDetectHost() string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "tailscale", "status", "--json").Output()
	if err != nil {
		return ""
	}
	return parseTailscaleStatus(out)
}

func parseTailscaleStatus(raw []byte) string {
	var doc struct {
		Self struct {
			DNSName string `json:"DNSName"`
		} `json:"Self"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return ""
	}
	return strings.TrimSuffix(doc.Self.DNSName, ".")
}
