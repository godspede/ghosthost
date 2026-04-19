package server

import (
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/godspede/ghosthost/internal/share"
)

type fakeLookup struct {
	sharesByToken map[string]*share.Share
}

func (f *fakeLookup) FindByToken(tok string) (*share.Share, bool) {
	s, ok := f.sharesByToken[tok]
	return s, ok
}
func (f *fakeLookup) MarkExpired(id, reason string) {}

func mkShare(t *testing.T, p string) *share.Share {
	t.Helper()
	return &share.Share{
		ID:          "abc",
		Token:       "t",
		SrcPath:     p,
		DisplayName: filepath.Base(p),
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(time.Hour),
	}
}

func TestServeBytes(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "img.png")
	content := []byte("PNGDATA")
	os.WriteFile(p, content, 0o644)

	l := &fakeLookup{sharesByToken: map[string]*share.Share{"t": mkShare(t, p)}}
	h := New(l)

	r := httptest.NewRequest("GET", "/t/t/img.png", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("code %d", w.Code)
	}
	body, _ := io.ReadAll(w.Body)
	if string(body) != string(content) {
		t.Fatalf("body mismatch: %q", body)
	}
	if w.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("bad cache-control: %q", w.Header().Get("Cache-Control"))
	}
	if w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("missing nosniff")
	}
}

func TestServe_UnknownToken_404(t *testing.T) {
	l := &fakeLookup{sharesByToken: map[string]*share.Share{}}
	h := New(l)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/t/xxx/a.mp4", nil))
	if w.Code != 404 {
		t.Fatalf("want 404, got %d", w.Code)
	}
	if w.Body.Len() != 0 {
		t.Fatalf("404 must have empty body")
	}
}

func TestServe_ExpiredShare_404(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "x")
	os.WriteFile(p, []byte{1}, 0o644)
	s := mkShare(t, p)
	s.ExpiresAt = time.Now().Add(-time.Minute)
	l := &fakeLookup{sharesByToken: map[string]*share.Share{"t": s}}
	h := New(l)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/t/t/x", nil))
	if w.Code != 404 {
		t.Fatalf("want 404, got %d", w.Code)
	}
}

func TestServe_MissingSource_404(t *testing.T) {
	s := mkShare(t, `C:\nope\does-not-exist`)
	l := &fakeLookup{sharesByToken: map[string]*share.Share{"t": s}}
	h := New(l)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/t/t/x", nil))
	if w.Code != 404 {
		t.Fatalf("want 404, got %d", w.Code)
	}
}

func TestServe_DlAttachment(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "img.png")
	os.WriteFile(p, []byte("x"), 0o644)
	l := &fakeLookup{sharesByToken: map[string]*share.Share{"t": mkShare(t, p)}}
	h := New(l)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/t/t/img.png?dl=1", nil))
	got := w.Header().Get("Content-Disposition")
	if got == "" || got[:len("attachment")] != "attachment" {
		t.Fatalf("want attachment disposition, got %q", got)
	}
}
