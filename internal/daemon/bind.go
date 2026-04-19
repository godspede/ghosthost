// internal/daemon/bind.go
package daemon

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"

	"github.com/godspede/ghosthost/internal/config"
)

func resolveBind(cfg config.Config) (string, error) {
	switch strings.ToLower(cfg.Bind) {
	case "0.0.0.0":
		slog.Warn("binding to 0.0.0.0 — server reachable on all networks the host joins")
		return "0.0.0.0", nil
	case "tailscale":
		ip, err := tailscaleIPv4()
		if err != nil {
			return "", fmt.Errorf("bind=tailscale requires working tailscale: %w", err)
		}
		return ip, nil
	default:
		return cfg.Bind, nil
	}
}

func tailscaleIPv4() (string, error) {
	out, err := exec.Command("tailscale", "status", "--json").Output()
	if err != nil {
		return "", err
	}
	var doc struct {
		Self struct {
			TailscaleIPs []string `json:"TailscaleIPs"`
		} `json:"Self"`
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		return "", err
	}
	for _, ip := range doc.Self.TailscaleIPs {
		if !strings.Contains(ip, ":") {
			return ip, nil
		}
	}
	return "", fmt.Errorf("no IPv4 tailscale address found")
}
