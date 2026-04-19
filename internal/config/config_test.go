// internal/config/config_test.go
package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoad_DefaultsWhenMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error when config missing (template written)")
	}
	if _, statErr := os.Stat(path); statErr != nil {
		t.Fatalf("expected template file written, got %v", statErr)
	}
}

func TestSaveAndLoadRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	c := Config{
		Host:         "homepc.tail-4a9c2e.ts.net",
		Bind:         "tailscale",
		Port:         8750,
		AdminPort:    8751,
		DataDir:      dir,
		DefaultTTL:   24 * time.Hour,
		IdleShutdown: 30 * time.Minute,
		TLSCert:      "/tmp/cert.pem",
		TLSKey:       "/tmp/key.pem",
	}
	if err := Save(path, c); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != c {
		t.Fatalf("roundtrip mismatch:\nwant %+v\ngot  %+v", c, got)
	}
}

func TestMergeOverrides(t *testing.T) {
	base := Config{Host: "a", Port: 8750}
	override := Config{Port: 9000}
	got := Merge(base, override)
	if got.Host != "a" {
		t.Errorf("lost base Host: %+v", got)
	}
	if got.Port != 9000 {
		t.Errorf("did not apply override Port: %+v", got)
	}
}

func TestLoad_RejectsMismatchedTLS(t *testing.T) {
	cases := map[string]string{
		"only-cert": `host = "h"` + "\n" + `tls_cert = "/tmp/cert.pem"`,
		"only-key":  `host = "h"` + "\n" + `tls_key = "/tmp/key.pem"`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "config.toml")
			if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := Load(path)
			if err == nil {
				t.Fatal("expected error for mismatched tls config")
			}
			msg := err.Error()
			if !contains(msg, "tls_cert") || !contains(msg, "tls_key") {
				t.Fatalf("error should mention both tls_cert and tls_key: %v", err)
			}
		})
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestLoad_RefusesEmptyHost(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(`host = ""`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for empty host")
	}
}
