// internal/daemon/run_test.go
package daemon

import (
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/godspede/ghosthost/internal/config"
)

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

func TestRun_StartAndStopViaIdle(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{
		Host:         "127.0.0.1",
		Bind:         "127.0.0.1",
		Port:         freePort(t),
		AdminPort:    freePort(t),
		DataDir:      dir,
		DefaultTTL:   time.Minute,
		IdleShutdown: 300 * time.Millisecond,
	}

	done := make(chan error, 1)
	go func() { done <- Run(cfg) }()

	lockPath := filepath.Join(dir, "daemon.lock")
	deadline := time.Now().Add(5 * time.Second)
	var meta Meta
	for {
		m, err := ReadLockfile(lockPath)
		if err == nil && m.Secret != "" {
			meta = m
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("daemon did not become ready")
		}
		time.Sleep(50 * time.Millisecond)
	}

	req, _ := http.NewRequest("GET", "http://127.0.0.1:"+strconv.Itoa(cfg.AdminPort)+"/status", nil)
	req.Header.Set("Authorization", "Bearer "+meta.Secret)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status code %d", resp.StatusCode)
	}
	resp.Body.Close()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("daemon did not exit on idle")
	}
}
