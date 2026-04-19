// internal/daemon/lock_other.go
//go:build !windows

package daemon

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"syscall"
)

var ErrLocked = errors.New("daemon already running")

type unixLock struct{ f *os.File }

func acquireLock(path string) (Lockfile, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		return nil, ErrLocked
	}
	return &unixLock{f: f}, nil
}

func (l *unixLock) Read() (Meta, error) {
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
func (l *unixLock) Write(m Meta) error {
	if err := l.f.Truncate(0); err != nil {
		return err
	}
	if _, err := l.f.Seek(0, io.SeekStart); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(m, "", "  ")
	if _, err := l.f.Write(b); err != nil {
		return err
	}
	return l.f.Sync()
}
func (l *unixLock) Close() error {
	_ = syscall.Flock(int(l.f.Fd()), syscall.LOCK_UN)
	return l.f.Close()
}

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
