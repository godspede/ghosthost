package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestRun_NoArgs_Usage(t *testing.T) {
	var out, errb bytes.Buffer
	code := Run(nil, &out, &errb)
	if code != ExitUsage {
		t.Fatalf("want ExitUsage, got %d", code)
	}
}

func TestRun_InvalidConfig_ExitConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(cfgPath, []byte("not a valid = toml = file"), 0o600); err != nil {
		t.Fatal(err)
	}
	var out, errb bytes.Buffer
	code := Run([]string{"--config", cfgPath, "list"}, &out, &errb)
	if code != ExitConfig {
		t.Fatalf("want ExitConfig, got %d (stderr=%s)", code, errb.String())
	}
}

func TestRun_UnknownCommand_Usage(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")
	body := `host = "127.0.0.1"
bind = "127.0.0.1"
port = 18750
admin_port = 18751
data_dir = ` + toTOMLString(dir) + `
default_ttl = "24h"
idle_shutdown = "30m"
`
	if err := os.WriteFile(cfgPath, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	var out, errb bytes.Buffer
	code := Run([]string{"--config", cfgPath, "bogus"}, &out, &errb)
	if code != ExitUsage {
		t.Fatalf("want ExitUsage, got %d (stderr=%s)", code, errb.String())
	}
}

func toTOMLString(s string) string {
	return "'" + s + "'"
}
