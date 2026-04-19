// internal/daemon/rotlog.go
package daemon

import (
	"io"
	"os"
	"sync"
)

const logRotateBytes = 5 * 1024 * 1024

type rotLog struct {
	path string
	mu   sync.Mutex
	f    *os.File
	size int64
}

func openRotatingLog(path string) (io.WriteCloser, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, err
	}
	info, _ := f.Stat()
	return &rotLog{path: path, f: f, size: info.Size()}, nil
}

func (r *rotLog) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.size+int64(len(p)) > logRotateBytes {
		_ = r.f.Close()
		_ = os.Rename(r.path, r.path+".1")
		f, err := os.OpenFile(r.path, os.O_CREATE|os.O_WRONLY, 0o600)
		if err != nil {
			return 0, err
		}
		r.f = f
		r.size = 0
	}
	n, err := r.f.Write(p)
	r.size += int64(n)
	return n, err
}

func (r *rotLog) Close() error { return r.f.Close() }
