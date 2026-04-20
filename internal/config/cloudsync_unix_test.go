//go:build !windows

package config

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestDetectCloudSync_EnvMatch on Unix-like systems uses POSIX absolute
// paths so filepath.Abs/HasPrefix work; mirrors the Windows test's intent.
func TestDetectCloudSync_EnvMatch(t *testing.T) {
	t.Setenv("OneDrive", "/home/x/OneDrive")
	reason := DetectCloudSync(filepath.Join("/home/x/OneDrive/Documents/gh"))
	if reason == "" || !strings.Contains(reason, "OneDrive") {
		t.Fatalf("want OneDrive reason, got %q", reason)
	}
}

func TestDetectCloudSync_NoMatch(t *testing.T) {
	t.Setenv("OneDrive", "")
	t.Setenv("DROPBOX", "")
	if r := DetectCloudSync("/tmp/gh"); r != "" {
		t.Fatalf("want empty, got %q", r)
	}
}
