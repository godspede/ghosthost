//go:build !windows

package daemon

import (
	"path/filepath"
	"testing"
)

func TestLock_SecondAcquireFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "daemon.lock")
	l1, err := AcquireLock(path)
	if err != nil {
		t.Fatal(err)
	}
	defer l1.Close()

	if _, err := AcquireLock(path); err != ErrLocked {
		t.Fatalf("want ErrLocked, got %v", err)
	}
}

func TestLock_ReleasedOnClose(t *testing.T) {
	path := filepath.Join(t.TempDir(), "daemon.lock")
	l1, _ := AcquireLock(path)
	l1.Close()
	l2, err := AcquireLock(path)
	if err != nil {
		t.Fatalf("expected re-acquire after close: %v", err)
	}
	l2.Close()
}

func TestLock_WriteAndReadMeta(t *testing.T) {
	path := filepath.Join(t.TempDir(), "daemon.lock")
	l, _ := AcquireLock(path)
	defer l.Close()
	meta := Meta{PID: 1234, AdminPort: 8751, Secret: "s", Version: "v"}
	if err := l.Write(meta); err != nil {
		t.Fatal(err)
	}
	got, err := ReadLockfile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != meta {
		t.Fatalf("mismatch: %+v vs %+v", got, meta)
	}
}
