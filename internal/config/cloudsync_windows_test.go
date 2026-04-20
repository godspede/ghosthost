//go:build windows

package config

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestDetectCloudSync_EnvMatch uses Windows-style absolute paths because the
// primary cloud-sync case (OneDrive env var populated) is Windows-specific;
// on other platforms filepath.Abs won't treat `C:\...` as absolute and the
// prefix match can't work.
func TestDetectCloudSync_EnvMatch(t *testing.T) {
	t.Setenv("OneDrive", `C:\Users\x\OneDrive`)
	reason := DetectCloudSync(filepath.Join(`C:\Users\x\OneDrive\Documents\gh`))
	if reason == "" || !strings.Contains(reason, "OneDrive") {
		t.Fatalf("want OneDrive reason, got %q", reason)
	}
}

func TestDetectCloudSync_NoMatch(t *testing.T) {
	t.Setenv("OneDrive", "")
	t.Setenv("DROPBOX", "")
	if r := DetectCloudSync(`C:\tmp\gh`); r != "" {
		t.Fatalf("want empty, got %q", r)
	}
}
