// internal/config/defaults.go
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	DefaultPort         = 8750
	DefaultAdminPort    = 8751
	DefaultBind         = "tailscale"
	DefaultTTL          = 24 * time.Hour
	DefaultIdleShutdown = 30 * time.Minute
)

func withDefaults(c Config) Config {
	if c.Bind == "" {
		c.Bind = DefaultBind
	}
	if c.Port == 0 {
		c.Port = DefaultPort
	}
	if c.AdminPort == 0 {
		c.AdminPort = DefaultAdminPort
	}
	if c.DataDir == "" {
		c.DataDir = defaultDataDir()
	}
	if c.DefaultTTL == 0 {
		c.DefaultTTL = DefaultTTL
	}
	if c.IdleShutdown == 0 {
		c.IdleShutdown = DefaultIdleShutdown
	}
	return c
}

func defaultDataDir() string {
	if la := os.Getenv("LOCALAPPDATA"); la != "" {
		return filepath.Join(la, "ghosthost")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ghosthost")
}

// DefaultConfigPath returns %APPDATA%\ghosthost\config.toml (or platform
// equivalent).
func DefaultConfigPath() string {
	if ad := os.Getenv("APPDATA"); ad != "" {
		return filepath.Join(ad, "ghosthost", "config.toml")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "ghosthost", "config.toml")
}

func writeTemplate(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	autoHost := autoDetectHost()
	tpl := fmt.Sprintf(`# ghosthost config
# Fill in 'host' with a reachable hostname (typically your Tailscale MagicDNS name).
host = %q

# bind: "tailscale" (default, resolves tailnet IP at startup), or an explicit IP.
# "0.0.0.0" exposes to all local networks; only set if you understand the risk.
bind = %q

port = %d
admin_port = %d
data_dir = %q
default_ttl = "24h"
idle_shutdown = "30m"

# Optional HTTPS. If both are set and readable, the public server uses TLS
# and URLs are https://. Leave blank for plain HTTP (fine on a trusted
# tailnet). For Tailscale users, `+"`"+`tailscale cert <magicdns-name>`+"`"+` produces
# a browser-trusted cert/key pair in the current directory.
# tls_cert = "C:\\Users\\you\\AppData\\Local\\ghosthost\\homepc.tail-4a9c2e.ts.net.crt"
# tls_key  = "C:\\Users\\you\\AppData\\Local\\ghosthost\\homepc.tail-4a9c2e.ts.net.key"
`, autoHost, DefaultBind, DefaultPort, DefaultAdminPort, defaultDataDir())
	return os.WriteFile(path, []byte(tpl), 0o600)
}
