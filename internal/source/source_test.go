package source

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolve_RegularFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Resolve(path)
	if err != nil {
		t.Fatalf("Resolve err: %v", err)
	}
	if !filepath.IsAbs(got) {
		t.Fatalf("Resolve returned non-abs: %q", got)
	}
}

func TestResolve_Missing(t *testing.T) {
	dir := t.TempDir()
	_, err := Resolve(filepath.Join(dir, "nope"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestResolve_Directory(t *testing.T) {
	_, err := Resolve(t.TempDir())
	if err == nil {
		t.Fatal("expected error for directory")
	}
}

func TestResolve_UNC(t *testing.T) {
	_, err := Resolve(`\\server\share\x.txt`)
	if err == nil || !strings.Contains(err.Error(), "UNC") {
		t.Fatalf("expected UNC rejection, got %v", err)
	}
}

func TestResolve_RawPath(t *testing.T) {
	for _, p := range []string{`\\?\C:\x.txt`, `\\.\PhysicalDrive0`} {
		if _, err := Resolve(p); err == nil {
			t.Errorf("expected rejection for %q", p)
		}
	}
}
