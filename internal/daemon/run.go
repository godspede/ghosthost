// internal/daemon/run.go
package daemon

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/godspede/ghosthost/internal/admin"
	"github.com/godspede/ghosthost/internal/config"
	"github.com/godspede/ghosthost/internal/history"
	"github.com/godspede/ghosthost/internal/server"
	"github.com/godspede/ghosthost/internal/version"
)

func Run(cfg config.Config) error {
	if err := os.MkdirAll(cfg.DataDir, 0o700); err != nil {
		return err
	}
	logPath := filepath.Join(cfg.DataDir, "daemon.log")
	logFile, err := openRotatingLog(logPath)
	if err != nil {
		return err
	}
	defer logFile.Close()
	slog.SetDefault(slog.New(slog.NewJSONHandler(logFile, nil)))
	slog.Info("daemon starting", "version", version.Version, "pid", os.Getpid(), "port", cfg.Port)

	if reason := config.DetectCloudSync(cfg.DataDir); reason != "" {
		slog.Warn("data_dir cloud-sync warning", "reason", reason)
	}

	lockPath := filepath.Join(cfg.DataDir, "daemon.lock")
	lock, err := AcquireLock(lockPath)
	if err != nil {
		return fmt.Errorf("acquire lock: %w", err)
	}
	defer lock.Close()

	secret, err := newSecret()
	if err != nil {
		return err
	}

	histPath := filepath.Join(cfg.DataDir, "history.jsonl")
	hist, err := history.Open(histPath)
	if err != nil {
		return err
	}
	defer hist.Close()

	ctx, cancel := context.WithCancel(context.Background())
	var stopOnce sync.Once
	stop := func() { stopOnce.Do(cancel) }

	core := NewCore(cfg, secret, hist, stop)
	if err := core.RestoreFromHistory(); err != nil {
		slog.Warn("history replay error", "err", err)
	}

	publicAddr, err := resolveBind(cfg)
	if err != nil {
		return fmt.Errorf("resolve bind: %w", err)
	}
	publicL, err := net.Listen("tcp", fmt.Sprintf("%s:%d", publicAddr, cfg.Port))
	if err != nil {
		return fmt.Errorf("listen public: %w", err)
	}
	defer publicL.Close()

	adminL, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", cfg.AdminPort))
	if err != nil {
		return fmt.Errorf("listen admin: %w", err)
	}
	defer adminL.Close()

	if err := lock.Write(Meta{
		PID:       os.Getpid(),
		AdminPort: cfg.AdminPort,
		Secret:    secret,
		Version:   version.Version,
	}); err != nil {
		return fmt.Errorf("write lock meta: %w", err)
	}

	adminHandler := touchAdminMiddleware(core, admin.NewHandler(core))

	publicSrv := &http.Server{Handler: server.New(core)}
	adminSrv := &http.Server{Handler: adminHandler}

	useTLS := cfg.TLSCert != "" && cfg.TLSKey != ""
	if useTLS {
		if err := checkTLSFilesReadable(cfg.TLSCert, cfg.TLSKey); err != nil {
			return fmt.Errorf("tls config: %w", err)
		}
	}

	go servePublic(publicSrv, publicL, cfg, useTLS, stop)
	go serveWithLog("admin", adminSrv, adminL, stop)

	go func() {
		t := time.NewTicker(30 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-t.C:
				core.ExpireDue(now)
			}
		}
	}()

	go func() {
		interval := 1 * time.Minute
		if cfg.IdleShutdown < interval {
			interval = cfg.IdleShutdown / 2
			if interval < 50*time.Millisecond {
				interval = 50 * time.Millisecond
			}
		}
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-t.C:
				idle, empty := core.IdleFor(now)
				if empty && idle >= cfg.IdleShutdown {
					slog.Info("idle shutdown", "idle_seconds", int(idle.Seconds()))
					stop()
					return
				}
			}
		}
	}()

	<-ctx.Done()
	shutdownCtx, sc := context.WithTimeout(context.Background(), 10*time.Second)
	defer sc()
	_ = publicSrv.Shutdown(shutdownCtx)
	_ = adminSrv.Shutdown(shutdownCtx)
	slog.Info("daemon stopped")
	return nil
}

func touchAdminMiddleware(core *Core, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.TouchAdmin()
		h.ServeHTTP(w, r)
	})
}

func newSecret() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func serveWithLog(name string, s *http.Server, l net.Listener, stop func()) {
	if err := s.Serve(l); err != nil && err != http.ErrServerClosed {
		slog.Error("server exited", "name", name, "err", err)
		stop()
	}
}

// checkTLSFilesReadable verifies the TLS cert and key paths are readable
// before we bind so we fail loudly at startup rather than in a later
// handshake.
func checkTLSFilesReadable(certPath, keyPath string) error {
	for label, p := range map[string]string{"tls_cert": certPath, "tls_key": keyPath} {
		f, err := os.Open(p)
		if err != nil {
			return fmt.Errorf("%s: %w", label, err)
		}
		f.Close()
	}
	// Parse as a PEM keypair so a bad file fails here, not on first request.
	if _, err := tls.LoadX509KeyPair(certPath, keyPath); err != nil {
		return fmt.Errorf("load keypair: %w", err)
	}
	return nil
}

func servePublic(s *http.Server, l net.Listener, cfg config.Config, useTLS bool, stop func()) {
	var err error
	if useTLS {
		err = s.ServeTLS(l, cfg.TLSCert, cfg.TLSKey)
	} else {
		err = s.Serve(l)
	}
	if err != nil && err != http.ErrServerClosed {
		slog.Error("public server exited", "err", err)
		stop()
	}
}
