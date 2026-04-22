// internal/daemon/daemon_test.go
package daemon

import (
	"errors"
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

func TestCore_Info(t *testing.T) {
	core, dir := newCore(t)
	p := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(p, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := core.Share(admin.ShareRequest{SrcPath: p, DisplayName: "hello.txt"})
	if err != nil {
		t.Fatal(err)
	}

	// lookup by token
	info, err := core.Info(got.Token)
	if err != nil {
		t.Fatalf("Info(token): %v", err)
	}
	if info.ID != got.ID {
		t.Errorf("Info(token).ID = %q, want %q", info.ID, got.ID)
	}
	if info.SrcPath == "" {
		t.Error("Info(token).SrcPath empty")
	}

	// lookup by id
	info, err = core.Info(got.ID)
	if err != nil {
		t.Fatalf("Info(id): %v", err)
	}
	if info.URL != got.URL {
		t.Errorf("Info(id).URL = %q, want %q", info.URL, got.URL)
	}

	// lookup by path-only URL
	info, err = core.Info("/t/" + got.Token + "/hello.txt")
	if err != nil {
		t.Fatalf("Info(path): %v", err)
	}
	if info.ID != got.ID {
		t.Errorf("Info(path).ID = %q, want %q", info.ID, got.ID)
	}

	// expired: clock moved past ExpiresAt before ExpireDue runs.
	// We need a separate share because the first one will be revoked below.
	p2 := filepath.Join(dir, "hello2.txt")
	if err := os.WriteFile(p2, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	got2, err := core.Share(admin.ShareRequest{SrcPath: p2, DisplayName: "hello2.txt"})
	if err != nil {
		t.Fatal(err)
	}
	fc := core.clock.(*fakeClock)
	fc.t = got2.ExpiresAt.Add(time.Second)
	if _, err := core.Info(got2.ID); !errors.Is(err, admin.ErrNotFound) {
		t.Errorf("Info on expired share: got %v, want admin.ErrNotFound", err)
	}
	// Restore the clock so subsequent assertions in this test (if any) aren't affected.
	fc.t = time.Unix(1700000000, 0)

	// unknown id
	if _, err := core.Info("zzzzzzzz"); !errors.Is(err, admin.ErrNotFound) {
		t.Errorf("Info(unknown id): got %v, want admin.ErrNotFound", err)
	}

	// revoked -> not found
	if err := core.Revoke(got.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := core.Info(got.ID); !errors.Is(err, admin.ErrNotFound) {
		t.Errorf("Info on revoked id: got %v, want admin.ErrNotFound", err)
	}
}
