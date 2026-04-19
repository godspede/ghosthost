// internal/daemon/daemon_test.go
package daemon

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/godspede/ghosthost/internal/admin"
	"github.com/godspede/ghosthost/internal/config"
	"github.com/godspede/ghosthost/internal/history"
)

type fakeClock struct{ t time.Time }

func (f *fakeClock) Now() time.Time { return f.t }

func newCore(t *testing.T) (*Core, string) {
	t.Helper()
	dir := t.TempDir()
	histPath := filepath.Join(dir, "history.jsonl")
	h, err := history.Open(histPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { h.Close() })
	cfg := config.Config{
		Host:         "host.example",
		Port:         8750,
		DefaultTTL:   time.Hour,
		IdleShutdown: 30 * time.Minute,
	}
	core := NewCore(cfg, "secret", h, func() {})
	core.clock = &fakeClock{t: time.Unix(1700000000, 0)}
	return core, dir
}

func TestShareAndList(t *testing.T) {
	core, dir := newCore(t)
	p := filepath.Join(dir, "a.txt")
	os.WriteFile(p, []byte("x"), 0o644)

	got, err := core.Share(admin.ShareRequest{SrcPath: p})
	if err != nil {
		t.Fatalf("share: %v", err)
	}
	if got.ID == "" || got.Token == "" {
		t.Fatal("missing id/token")
	}
	list := core.List()
	if len(list.Shares) != 1 || list.Shares[0].ID != got.ID {
		t.Fatalf("list mismatch: %+v", list)
	}
}

func TestRevoke(t *testing.T) {
	core, dir := newCore(t)
	p := filepath.Join(dir, "a.txt")
	os.WriteFile(p, []byte("x"), 0o644)
	g, _ := core.Share(admin.ShareRequest{SrcPath: p})
	if err := core.Revoke(g.ID); err != nil {
		t.Fatal(err)
	}
	if len(core.List().Shares) != 0 {
		t.Fatal("expected empty list after revoke")
	}
}

func TestReshare_NewTokenSameName(t *testing.T) {
	core, dir := newCore(t)
	p := filepath.Join(dir, "a.txt")
	os.WriteFile(p, []byte("x"), 0o644)
	orig, _ := core.Share(admin.ShareRequest{SrcPath: p, DisplayName: "a.txt"})
	re, err := core.Reshare(orig.ID)
	if err != nil {
		t.Fatal(err)
	}
	if re.Token == orig.Token {
		t.Fatal("reshare must produce new token")
	}
	if re.ID == orig.ID {
		t.Fatal("reshare must produce new id")
	}
}

func TestExpireDue(t *testing.T) {
	core, dir := newCore(t)
	p := filepath.Join(dir, "a.txt")
	os.WriteFile(p, []byte("x"), 0o644)
	core.Share(admin.ShareRequest{SrcPath: p, TTLSeconds: 1})
	fc := core.clock.(*fakeClock)
	fc.t = fc.t.Add(2 * time.Second)
	core.ExpireDue(fc.t)
	if len(core.List().Shares) != 0 {
		t.Fatal("expected share expired")
	}
}
