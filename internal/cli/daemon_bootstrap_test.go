package cli

import (
	"os"
	"strings"
	"testing"
)

func TestIsTestBinary(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"/usr/local/bin/ghosthost", false},
		{"C:\\Program Files\\ghosthost.exe", false},
		{"/tmp/go-build1234/cli.test", true},
		{"C:\\Users\\x\\AppData\\Local\\Temp\\go-build\\cli.test.exe", true},
		{"/tmp/CLI.TEST", true},
		{"C:\\path\\CLI.TEST.EXE", true},
	}
	for _, c := range cases {
		if got := isTestBinary(c.path); got != c.want {
			t.Errorf("isTestBinary(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}

// When invoked from `go test`, os.Executable() points at the test runner.
// EnsureDaemon must refuse to spawn it detached; otherwise a zombie test
// binary survives the run and holds open handles on its t.TempDir trees,
// spamming the OS temp directory on Windows.
func TestEnsureDaemon_RefusesToSpawnTestBinary(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	if !isTestBinary(self) {
		t.Skipf("executable %q not recognized as a test binary; guard cannot be exercised here", self)
	}

	dataDir := t.TempDir()
	_, err = EnsureDaemon(dataDir, "")
	if err == nil {
		t.Fatal("EnsureDaemon returned nil error; expected refusal to spawn from test binary")
	}
	if !strings.Contains(err.Error(), "refusing to spawn daemon from test binary") {
		t.Errorf("error = %q, want substring %q", err.Error(), "refusing to spawn daemon from test binary")
	}
}
