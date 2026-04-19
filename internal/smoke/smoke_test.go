// internal/smoke/smoke_test.go
//go:build smoke

package smoke_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func build(t *testing.T) string {
	t.Helper()
	binName := "ghosthost"
	if runtime.GOOS == "windows" {
		binName = "ghosthost.exe"
	}
	bin := filepath.Join(t.TempDir(), binName)
	cmd := exec.Command("go", "build", "-o", bin, "../../cmd/ghosthost")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("build: %v", err)
	}
	return bin
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	p := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return p
}

func writeConfig(t *testing.T, dir string, port, adminPort int) string {
	t.Helper()
	path := filepath.Join(dir, "config.toml")
	dataDirJSON, _ := json.Marshal(dir)
	body := fmt.Sprintf(`
host = "127.0.0.1"
bind = "127.0.0.1"
port = %d
admin_port = %d
data_dir = %s
default_ttl = "1h"
idle_shutdown = "10m"
`, port, adminPort, dataDirJSON)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestSmoke_ShareAndFetch(t *testing.T) {
	if os.Getenv("GH_SMOKE") != "1" {
		t.Skip("set GH_SMOKE=1 to run")
	}
	dir := t.TempDir()
	bin := build(t)
	port, adminPort := freePort(t), freePort(t)
	cfgPath := writeConfig(t, dir, port, adminPort)

	payloadPath := filepath.Join(dir, "clip.txt")
	content := []byte("smokecontent")
	if err := os.WriteFile(payloadPath, content, 0o644); err != nil {
		t.Fatal(err)
	}

	// Ensure daemon is stopped at test end regardless of outcome.
	defer func() {
		_ = exec.Command(bin, "--config", cfgPath, "stop").Run()
	}()

	// share — relies on CLI auto-spawn, which forwards --config to the
	// daemon so the child loads the same config as the CLI.
	var out bytes.Buffer
	cmd := exec.Command(bin, "--config", cfgPath, "--json", "share", payloadPath)
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("share failed: %v", err)
	}
	var p struct {
		URL string `json:"url"`
		ID  string `json:"id"`
	}
	if err := json.Unmarshal(out.Bytes(), &p); err != nil {
		t.Fatalf("parse share output: %v: %q", err, out.String())
	}

	// fetch
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", p.URL, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 || !bytes.Equal(b, content) {
		t.Fatalf("fetch failed: status=%d body=%q", resp.StatusCode, string(b))
	}

	// revoke
	out.Reset()
	cmd = exec.Command(bin, "--config", cfgPath, "revoke", p.ID)
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if !strings.Contains(out.String(), "ok") {
		t.Fatalf("revoke output: %q", out.String())
	}

	// fetch again -> 404
	resp2, err := http.Get(p.URL)
	if err != nil {
		t.Fatal(err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != 404 {
		t.Fatalf("post-revoke status %d", resp2.StatusCode)
	}
}
