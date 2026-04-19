// internal/history/history.go
package history

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
)

// Store is a single-writer append-only JSONL history store.
type Store struct {
	path string
	mu   sync.Mutex
	f    *os.File
}

// Open creates or opens the history file, truncating any trailing partial
// line that might be left over from a crash.
func Open(path string) (*Store, error) {
	if err := truncatePartialTrailingLine(path); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open history: %w", err)
	}
	return &Store{path: path, f: f}, nil
}

func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.f == nil {
		return nil
	}
	err := s.f.Close()
	s.f = nil
	return err
}

// Append serializes ev as JSON, appends a line, and fsyncs.
func (s *Store) Append(ev Event) error {
	line, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	line = append(line, '\n')

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.f.Write(line); err != nil {
		return fmt.Errorf("write event: %w", err)
	}
	return s.f.Sync()
}

// Replay returns all events in file order. Corrupt interior lines are
// skipped with a warning; trailing partial lines were already truncated
// in Open.
func (s *Store) Replay() ([]Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.f.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	defer s.f.Seek(0, io.SeekEnd)

	var out []Event
	sc := bufio.NewScanner(s.f)
	sc.Buffer(make([]byte, 1<<16), 1<<20)
	lineNum := 0
	for sc.Scan() {
		lineNum++
		var ev Event
		if err := json.Unmarshal(sc.Bytes(), &ev); err != nil {
			slog.Warn("history: skipping corrupt line", "path", s.path, "line", lineNum, "err", err)
			continue
		}
		out = append(out, ev)
	}
	if err := sc.Err(); err != nil {
		return out, fmt.Errorf("scan: %w", err)
	}
	return out, nil
}

// truncatePartialTrailingLine ensures the file either doesn't exist, is
// empty, or ends with a newline. A partial trailing line is removed.
func truncatePartialTrailingLine(path string) error {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Size() == 0 {
		return nil
	}
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer f.Close()

	const chunk = 4096
	pos := info.Size()
	for pos > 0 {
		readAt := pos - chunk
		n := int64(chunk)
		if readAt < 0 {
			n += readAt
			readAt = 0
		}
		buf := make([]byte, n)
		if _, err := f.ReadAt(buf, readAt); err != nil {
			return err
		}
		for i := int(n) - 1; i >= 0; i-- {
			if buf[i] == '\n' {
				keep := readAt + int64(i) + 1
				if keep == info.Size() {
					return nil
				}
				return f.Truncate(keep)
			}
		}
		pos = readAt
	}
	return f.Truncate(0)
}
