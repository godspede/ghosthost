// internal/config/cloudsync_test.go
package config

import (
	"path/filepath"
	"strings"
	"testing"
)

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
