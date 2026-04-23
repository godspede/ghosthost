// internal/cli/daemon_bootstrap.go
package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/godspede/ghosthost/internal/daemon"
)

func isTestBinary(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	return strings.HasSuffix(base, ".test") || strings.HasSuffix(base, ".test.exe")
}

// EnsureDaemon makes sure the daemon is running and returns an admin
// client wired to it. If the daemon is not running, spawns it (forwarding
// cfgPath via --config so the child loads the same config as the CLI) and
// waits up to 2 seconds for readiness.
func EnsureDaemon(dataDir, cfgPath string) (*Client, error) {
	lockPath := filepath.Join(dataDir, "daemon.lock")
	if c, err := tryConnect(lockPath); err == nil {
		return c, nil
	}
	self, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("locate self: %w", err)
	}
	// Refuse to daemonize a test binary. Under `go test`, os.Executable()
	// points at the test runner (e.g. cli.test.exe); spawning it detached
	// leaks a zombie that holds open handles on its t.TempDir trees, which
	// on Windows blocks cleanup and spams %TEMP%\Test<Name>* forever.
	if isTestBinary(self) {
		return nil, fmt.Errorf("refusing to spawn daemon from test binary %q", self)
	}
	var extraArgs []string
	if cfgPath != "" {
		extraArgs = []string{"--config", cfgPath}
	}
	if err := daemon.SpawnDetached(self, extraArgs...); err != nil {
		return nil, fmt.Errorf("spawn daemon: %w", err)
	}
	deadline := time.Now().Add(10 * time.Second)
	for {
		if c, err := tryConnect(lockPath); err == nil {
			return c, nil
		}
		if time.Now().After(deadline) {
			return nil, ErrUnreachable
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func tryConnect(lockPath string) (*Client, error) {
	meta, err := daemon.ReadLockfile(lockPath)
	if err != nil {
		return nil, err
	}
	if meta.AdminPort == 0 || meta.Secret == "" {
		return nil, errors.New("lockfile incomplete")
	}
	c := NewClient(meta.AdminPort, meta.Secret)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	if _, err := c.Status(ctx); err != nil {
		return nil, err
	}
	return c, nil
}
