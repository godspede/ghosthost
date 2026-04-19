// internal/daemon/lock_windows.go
//go:build windows

package daemon

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

// ErrLocked is returned when another process holds the lockfile.
var ErrLocked = errors.New("daemon already running")

type winLock struct{ f *os.File }

func acquireLock(path string) (Lockfile, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	handle := windows.Handle(f.Fd())
	ov := windows.Overlapped{OffsetHigh: 0xFFFFFFFF, Offset: 0xFFFFFFFF}
	err = windows.LockFileEx(handle,
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
		0, 1, 0, &ov)
	if err != nil {
		f.Close()
		return nil, ErrLocked
	}
	return &winLock{f: f}, nil
}

func (l *winLock) Read() (Meta, error) {
	if _, err := l.f.Seek(0, io.SeekStart); err != nil {
		return Meta{}, err
	}
	var m Meta
	dec := json.NewDecoder(l.f)
	if err := dec.Decode(&m); err != nil && err != io.EOF {
		return Meta{}, err
	}
	return m, nil
}

func (l *winLock) Write(m Meta) error {
	if err := l.f.Truncate(0); err != nil {
		return err
	}
	if _, err := l.f.Seek(0, io.SeekStart); err != nil {
		return err
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	if _, err := l.f.Write(b); err != nil {
		return err
	}
	_ = restrictACLOwnerOnly(l.f.Name())
	return l.f.Sync()
}

func (l *winLock) Close() error {
	handle := windows.Handle(l.f.Fd())
	ov := windows.Overlapped{OffsetHigh: 0xFFFFFFFF, Offset: 0xFFFFFFFF}
	_ = windows.UnlockFileEx(handle, 0, 1, 0, &ov)
	return l.f.Close()
}

// ReadLockfile reads lockfile metadata without holding the lock (for CLIs).
func ReadLockfile(path string) (Meta, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Meta{}, err
	}
	var m Meta
	if err := json.Unmarshal(b, &m); err != nil {
		return Meta{}, err
	}
	return m, nil
}
