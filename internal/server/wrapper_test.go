package server

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/godspede/ghosthost/internal/share"
)

func TestWrapper_ServedForVideoHTMLAccept(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "clip.mp4")
	os.WriteFile(p, []byte("fakevideo"), 0o644)
	s := mkShare(t, p)
	s.DisplayName = "clip.mp4"

	l := &fakeLookup{sharesByToken: map[string]*share.Share{"t": s}}
	h := New(l)
	r := httptest.NewRequest("GET", "/t/t/clip.mp4", nil)
	r.Header.Set("Accept", "text/html,*/*;q=0.8")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("want html content-type, got %q", ct)
	}
	body := w.Body.String()
	if !strings.Contains(body, `<video src="?raw=1"`) {
		t.Fatalf("wrapper missing <video>: %s", body)
	}
	if !strings.Contains(body, "muted") || !strings.Contains(body, "autoplay") {
		t.Fatalf("wrapper missing muted/autoplay")
	}
	if !strings.Contains(body, "max-width") {
		t.Fatalf("video wrapper missing max-width: %s", body)
	}
	if !strings.Contains(body, "playsinline") {
		t.Fatalf("video wrapper missing playsinline: %s", body)
	}
	// Regression guard: controlslist=nodownload was tried and removed. In
	// Firefox on desktop it hid the mute/unmute toggle (likely overriding
	// the volume control too). The cosmetic "no download button" win wasn't
	// worth losing cross-browser functional controls; ?dl=1 remains the
	// explicit download path.
	if strings.Contains(body, "controlslist") {
		t.Fatalf("video wrapper should not set controlslist (breaks Firefox controls): %s", body)
	}
	// Regression guard: the fix relies on 100dvh to keep controls inside the
	// real visible viewport on mobile. If max-height:100dvh disappears, the
	// fix is gone.
	if !strings.Contains(body, "max-height:100dvh") {
		t.Fatalf("video wrapper must use max-height:100dvh for mobile controls fix: %s", body)
	}
	// Regression guard: video element itself must not be sized to full viewport
	// height (which hides the native controls under mobile browser chrome).
	// We allow `max-height:100vh` (fallback for dvh) but not bare `height:100vh`.
	scrubbed := strings.ReplaceAll(body, "max-height:100vh", "")
	scrubbed = strings.ReplaceAll(scrubbed, "min-height:100vh", "")
	if strings.Contains(scrubbed, "height:100vh") {
		t.Fatalf("video wrapper must not set bare height:100vh on video element: %s", body)
	}
}

func TestWrapper_BypassedOnRaw(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "clip.mp4")
	os.WriteFile(p, []byte("fakevideo"), 0o644)
	s := mkShare(t, p)
	s.DisplayName = "clip.mp4"
	l := &fakeLookup{sharesByToken: map[string]*share.Share{"t": s}}
	h := New(l)
	r := httptest.NewRequest("GET", "/t/t/clip.mp4?raw=1", nil)
	r.Header.Set("Accept", "text/html")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if strings.HasPrefix(w.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("raw=1 should bypass wrapper")
	}
}

func TestWrapper_BypassedOnRange(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "clip.mp4")
	os.WriteFile(p, []byte("fakevideo"), 0o644)
	s := mkShare(t, p)
	s.DisplayName = "clip.mp4"
	l := &fakeLookup{sharesByToken: map[string]*share.Share{"t": s}}
	h := New(l)
	r := httptest.NewRequest("GET", "/t/t/clip.mp4", nil)
	r.Header.Set("Accept", "text/html")
	r.Header.Set("Range", "bytes=0-3")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if strings.HasPrefix(w.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("Range should bypass wrapper")
	}
}

func TestWrapper_NonVideoNever(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "img.png")
	os.WriteFile(p, []byte("x"), 0o644)
	s := mkShare(t, p)
	s.DisplayName = "img.png"
	l := &fakeLookup{sharesByToken: map[string]*share.Share{"t": s}}
	h := New(l)
	r := httptest.NewRequest("GET", "/t/t/img.png", nil)
	r.Header.Set("Accept", "text/html")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if strings.HasPrefix(w.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("non-video must not get wrapper")
	}
}

// .mkv's MIME isn't always registered in Windows' registry; the package-local
// MIME table must still steer us into the wrapper path.
func TestWrapper_VideoMKV(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "clip.mkv")
	os.WriteFile(p, []byte("fakemkv"), 0o644)
	s := mkShare(t, p)
	s.DisplayName = "clip.mkv"
	l := &fakeLookup{sharesByToken: map[string]*share.Share{"t": s}}
	h := New(l)
	r := httptest.NewRequest("GET", "/t/t/clip.mkv", nil)
	r.Header.Set("Accept", "text/html,*/*;q=0.8")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("want html wrapper for .mkv, got %q", ct)
	}
	if !strings.Contains(w.Body.String(), "<video") {
		t.Fatalf("mkv wrapper missing <video>")
	}
}

func TestWrapper_AudioHTMLAccept(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "clip.mp3")
	os.WriteFile(p, []byte("fakemp3"), 0o644)
	s := mkShare(t, p)
	s.DisplayName = "clip.mp3"
	l := &fakeLookup{sharesByToken: map[string]*share.Share{"t": s}}
	h := New(l)
	r := httptest.NewRequest("GET", "/t/t/clip.mp3", nil)
	r.Header.Set("Accept", "text/html,*/*;q=0.8")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("want html content-type, got %q", ct)
	}
	body := w.Body.String()
	if !strings.Contains(body, "<audio") {
		t.Fatalf("audio wrapper missing <audio>: %s", body)
	}
	if !strings.Contains(body, "autoplay") || !strings.Contains(body, "controls") {
		t.Fatalf("audio wrapper missing autoplay/controls: %s", body)
	}
	if strings.Contains(body, "muted") {
		t.Fatalf("audio wrapper must NOT contain muted: %s", body)
	}
	if strings.Contains(body, "<video") {
		t.Fatalf("audio wrapper must NOT contain <video>: %s", body)
	}
	if !strings.Contains(body, "width:") && !strings.Contains(body, "max-width:") {
		t.Fatalf("audio wrapper missing width/max-width: %s", body)
	}
	// Regression guard: audio element must not collapse to the browser default
	// tiny width on desktop — keep an explicit max-width.
	if !strings.Contains(body, "max-width:560px") {
		t.Fatalf("audio wrapper missing max-width:560px regression guard: %s", body)
	}
	if strings.Count(body, "clip.mp3") < 2 {
		t.Fatalf("audio filename should appear at least twice (title + label): %s", body)
	}
	// Ensure the name is visible in the body, not just inside <title>. The
	// body label lets users see which track is playing.
	titleIdx := strings.Index(body, "<title>")
	titleEnd := strings.Index(body, "</title>")
	if titleIdx < 0 || titleEnd < 0 {
		t.Fatalf("audio wrapper missing <title>: %s", body)
	}
	bodyWithoutTitle := body[:titleIdx] + body[titleEnd+len("</title>"):]
	if !strings.Contains(bodyWithoutTitle, "clip.mp3") {
		t.Fatalf("audio wrapper must show filename in body, not just title: %s", body)
	}
}

func TestWrapper_AudioBypassedOnRaw(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "clip.mp3")
	os.WriteFile(p, []byte("fakemp3"), 0o644)
	s := mkShare(t, p)
	s.DisplayName = "clip.mp3"
	l := &fakeLookup{sharesByToken: map[string]*share.Share{"t": s}}
	h := New(l)
	r := httptest.NewRequest("GET", "/t/t/clip.mp3?raw=1", nil)
	r.Header.Set("Accept", "text/html")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if strings.HasPrefix(w.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("raw=1 should bypass audio wrapper")
	}
	if w.Body.String() != "fakemp3" {
		t.Fatalf("want raw bytes, got %q", w.Body.String())
	}
}

func TestWrapper_AudioBypassedOnRange(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "clip.mp3")
	os.WriteFile(p, []byte("fakemp3"), 0o644)
	s := mkShare(t, p)
	s.DisplayName = "clip.mp3"
	l := &fakeLookup{sharesByToken: map[string]*share.Share{"t": s}}
	h := New(l)
	r := httptest.NewRequest("GET", "/t/t/clip.mp3", nil)
	r.Header.Set("Accept", "text/html")
	r.Header.Set("Range", "bytes=0-3")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if strings.HasPrefix(w.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("Range should bypass audio wrapper")
	}
}

func TestWrapper_PDFNoWrapper(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "doc.pdf")
	os.WriteFile(p, []byte("%PDF-1.4"), 0o644)
	s := mkShare(t, p)
	s.DisplayName = "doc.pdf"
	l := &fakeLookup{sharesByToken: map[string]*share.Share{"t": s}}
	h := New(l)
	r := httptest.NewRequest("GET", "/t/t/doc.pdf", nil)
	r.Header.Set("Accept", "text/html")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if strings.HasPrefix(w.Header().Get("Content-Type"), "text/html") {
		t.Fatalf(".pdf must not get wrapper, got ct=%q", w.Header().Get("Content-Type"))
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/pdf" {
		t.Fatalf("want application/pdf, got %q", ct)
	}
}

func TestWrapper_TxtNoWrapper(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "readme.txt")
	os.WriteFile(p, []byte("hello"), 0o644)
	s := mkShare(t, p)
	s.DisplayName = "readme.txt"
	l := &fakeLookup{sharesByToken: map[string]*share.Share{"t": s}}
	h := New(l)
	r := httptest.NewRequest("GET", "/t/t/readme.txt", nil)
	r.Header.Set("Accept", "text/html")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	// text/plain is expected — not our text/html wrapper.
	ct := w.Header().Get("Content-Type")
	if strings.HasPrefix(ct, "text/html") {
		t.Fatalf(".txt must not get html wrapper, got %q", ct)
	}
	if !strings.HasPrefix(ct, "text/plain") {
		t.Fatalf("want text/plain, got %q", ct)
	}
}

func TestWrapper_EscapesHostileName(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "clip.mp4")
	os.WriteFile(p, []byte("x"), 0o644)
	s := mkShare(t, p)
	s.DisplayName = `</title><script>alert(1)</script>.mp4`

	l := &fakeLookup{sharesByToken: map[string]*share.Share{"t": s}}
	h := New(l)
	r := httptest.NewRequest("GET", "/t/t/anything", nil)
	r.Header.Set("Accept", "text/html")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if strings.Contains(w.Body.String(), "<script>") {
		t.Fatalf("html escaping failed: %s", w.Body.String())
	}
}
