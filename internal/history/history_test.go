// internal/history/history_test.go
package history

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newStore(t *testing.T) (*Store, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "history.jsonl")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s, path
}

func TestAppendAndReplay(t *testing.T) {
	s, path := newStore(t)
	now := time.Now().UTC().Truncate(time.Second)
	digest := make([]byte, 32)
	digest[0] = 0xab
	shareEv := Event{
		Op:        OpShare,
		ID:        "id1",
		TokenHash: hex.EncodeToString(digest),
		Src:       `C:\x.mp4`,
		Name:      "x.mp4",
		CreatedAt: now,
		ExpiresAt: now.Add(24 * time.Hour),
	}
	if err := s.Append(shareEv); err != nil {
		t.Fatal(err)
	}
	if err := s.Append(Event{Op: OpRevoke, ID: "id1", At: now.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	s.Close()

	s2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()

	events, err := s2.Replay()
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("want 2 events, got %d", len(events))
	}
	if events[0].Op != OpShare || events[1].Op != OpRevoke {
		t.Fatalf("bad order: %+v", events)
	}
}

func TestReplay_TruncatesPartialTrailingLine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.jsonl")
	content := `{"op":"share","id":"id1"}` + "\n" + `{"op":"revo`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	events, err := s.Replay()
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].ID != "id1" {
		t.Fatalf("want single valid event, got %+v", events)
	}
	info, _ := os.Stat(path)
	want := int64(len(`{"op":"share","id":"id1"}` + "\n"))
	if info.Size() != want {
		t.Fatalf("want truncated to %d bytes, got %d", want, info.Size())
	}
}

func TestReplay_SkipsCorruptInteriorLine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.jsonl")
	content := `{"op":"share","id":"a"}` + "\n" + `GARBAGE` + "\n" + `{"op":"share","id":"b"}` + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	events, err := s.Replay()
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[0].ID != "a" || events[1].ID != "b" {
		t.Fatalf("corrupt line not skipped correctly: %+v", events)
	}
}

func TestAppend_NeverWritesRawToken(t *testing.T) {
	s, path := newStore(t)
	if err := s.Append(Event{Op: OpShare, ID: "x", TokenHash: "deadbeef"}); err != nil {
		t.Fatal(err)
	}
	s.Close()
	b, _ := os.ReadFile(path)
	for _, forbidden := range []string{`"token":`, `"token" `} {
		if strings.Contains(string(b), forbidden) {
			t.Fatalf("raw token field present in %q", string(b))
		}
	}
}
