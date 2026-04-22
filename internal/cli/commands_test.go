package cli

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/godspede/ghosthost/internal/admin"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func newTestOpts(t *testing.T) (*globalOpts, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	var out, errb bytes.Buffer
	o := &globalOpts{
		stdout: &out,
		stderr: &errb,
		format: Human,
		// cfg intentionally zero — tests that need daemon will short-circuit
		// before EnsureDaemon if validation fails first.
	}
	return o, &out, &errb
}

func writeTempFile(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "f-"+randHex()+".txt")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func randHex() string {
	var b [4]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func TestPrintShare_MinimalDefault(t *testing.T) {
	var buf bytes.Buffer
	p := admin.SharePayload{
		SchemaVersion: "1",
		ID:            "abc12345",
		Token:         "t",
		URL:           "http://h/t/t/x",
		ExpiresAt:     time.Unix(2_000_000_000, 0),
	}
	printShare(&buf, Human, p, false)
	got := buf.String()
	want := "http://h/t/t/x\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestPrintShare_Verbose(t *testing.T) {
	var buf bytes.Buffer
	p := admin.SharePayload{URL: "http://h/t/t/x", ID: "abc12345", ExpiresAt: time.Unix(2_000_000_000, 0)}
	printShare(&buf, Human, p, true)
	got := buf.String()
	if !strings.Contains(got, "URL:") || !strings.Contains(got, "ID:") || !strings.Contains(got, "Expires:") {
		t.Errorf("verbose output missing fields: %q", got)
	}
}

func TestPrintShare_JSONUnchanged(t *testing.T) {
	p := admin.SharePayload{SchemaVersion: "1", ID: "abc12345", Token: "tok", URL: "u"}

	// verbose=false, JSON format → bare object, no array
	var buf bytes.Buffer
	printShare(&buf, JSON, p, false)
	if !strings.Contains(buf.String(), `"id":"abc12345"`) {
		t.Errorf("JSON output missing id: %q", buf.String())
	}
	if strings.HasPrefix(strings.TrimSpace(buf.String()), "[") {
		t.Errorf("PR1 JSON should be a bare object, got array: %q", buf.String())
	}

	// verbose=true, JSON format → verbose flag must be ignored; same JSON output
	var bufV bytes.Buffer
	printShare(&bufV, JSON, p, true)
	if bufV.String() != buf.String() {
		t.Errorf("verbose should not affect JSON output.\nverbose=false: %q\nverbose=true:  %q", buf.String(), bufV.String())
	}
	if strings.Contains(bufV.String(), "URL:") {
		t.Error("verbose JSON output accidentally included human-block labels")
	}
}

func TestPrintInfo_Human(t *testing.T) {
	var buf bytes.Buffer
	p := admin.InfoPayload{
		SharePayload: admin.SharePayload{
			SchemaVersion: "1",
			ID:            "abc12345",
			URL:           "http://h/t/tok/x",
			ExpiresAt:     time.Unix(2_000_000_000, 0),
		},
		SrcPath:   "/abs/x",
		CreatedAt: time.Unix(1_000_000_000, 0),
	}
	printInfo(&buf, Human, p)
	out := buf.String()
	for _, s := range []string{"URL:", "ID:", "Src:", "Created:", "Expires:", "abc12345", "/abs/x"} {
		if !strings.Contains(out, s) {
			t.Errorf("missing %q in: %q", s, out)
		}
	}
}

func TestPrintInfo_JSON(t *testing.T) {
	var buf bytes.Buffer
	p := admin.InfoPayload{
		SharePayload: admin.SharePayload{SchemaVersion: "1", ID: "abc12345"},
		SrcPath:      "/abs/x",
	}
	printInfo(&buf, JSON, p)
	if !strings.Contains(buf.String(), `"src_path":"/abs/x"`) {
		t.Errorf("missing src_path: %q", buf.String())
	}
}

func TestPrintShares_DefaultMultiple(t *testing.T) {
	var buf bytes.Buffer
	payloads := []admin.SharePayload{
		{URL: "http://h/t/a/one.png"},
		{URL: "http://h/t/b/two.png"},
		{URL: "http://h/t/c/three.png"},
	}
	printShares(&buf, Human, payloads, false)
	want := "http://h/t/a/one.png\nhttp://h/t/b/two.png\nhttp://h/t/c/three.png\n"
	if buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}

func TestPrintShares_DefaultSingle(t *testing.T) {
	var buf bytes.Buffer
	payloads := []admin.SharePayload{{URL: "http://h/t/a/x.png"}}
	printShares(&buf, Human, payloads, false)
	if buf.String() != "http://h/t/a/x.png\n" {
		t.Errorf("got %q", buf.String())
	}
}

func TestPrintShares_Verbose(t *testing.T) {
	var buf bytes.Buffer
	payloads := []admin.SharePayload{
		{URL: "http://h/t/a/one.png", ID: "aaa11111", ExpiresAt: time.Unix(2_000_000_000, 0)},
		{URL: "http://h/t/b/two.png", ID: "bbb22222", ExpiresAt: time.Unix(2_000_000_000, 0)},
	}
	printShares(&buf, Human, payloads, true)
	out := buf.String()
	if strings.Count(out, "URL:") != 2 {
		t.Errorf("want 2 URL: lines, got: %q", out)
	}
	if !strings.Contains(out, "aaa11111") || !strings.Contains(out, "bbb22222") {
		t.Errorf("missing ids: %q", out)
	}
}

func TestPrintShares_JSONIsArray(t *testing.T) {
	var buf bytes.Buffer
	payloads := []admin.SharePayload{{SchemaVersion: "1", ID: "aaa11111"}}
	printShares(&buf, JSON, payloads, false)
	s := strings.TrimSpace(buf.String())
	if !strings.HasPrefix(s, "[") || !strings.HasSuffix(s, "]") {
		t.Errorf("want JSON array, got: %q", s)
	}
	if !strings.Contains(s, `"id":"aaa11111"`) {
		t.Errorf("missing id in array: %q", s)
	}
}

func TestPrintShares_JSONArrayEvenForSingle(t *testing.T) {
	// PR2 breaking change: single-file --json is now a one-element array.
	var buf bytes.Buffer
	payloads := []admin.SharePayload{{SchemaVersion: "1", ID: "only"}}
	printShares(&buf, JSON, payloads, false)
	s := strings.TrimSpace(buf.String())
	if !strings.HasPrefix(s, "[") {
		t.Errorf("single-file --json must be an array, got: %q", s)
	}
}

// ---------------------------------------------------------------------------
// cmdShare validation tests
// ---------------------------------------------------------------------------

func TestCmdShare_NoArgs(t *testing.T) {
	o, _, errb := newTestOpts(t)
	code := cmdShare(context.Background(), nil, o)
	if code != ExitUsage {
		t.Errorf("code = %d, want ExitUsage", code)
	}
	if !strings.Contains(errb.String(), "usage:") {
		t.Errorf("stderr missing usage: %q", errb.String())
	}
}

func TestCmdShare_AsWithMultiple_Rejected(t *testing.T) {
	o, _, errb := newTestOpts(t)
	a := writeTempFile(t, "a")
	b := writeTempFile(t, "b")
	code := cmdShare(context.Background(), []string{"--as", "x", a, b}, o)
	if code != ExitUsage {
		t.Errorf("code = %d, want ExitUsage", code)
	}
	if !strings.Contains(errb.String(), "--as") {
		t.Errorf("stderr missing --as hint: %q", errb.String())
	}
}

func TestCmdShare_CapWithoutYes_Rejected(t *testing.T) {
	o, _, errb := newTestOpts(t)
	args := []string{}
	for i := 0; i < 65; i++ {
		args = append(args, writeTempFile(t, "x"))
	}
	code := cmdShare(context.Background(), args, o)
	if code != ExitUsage {
		t.Errorf("code = %d, want ExitUsage", code)
	}
	if !strings.Contains(errb.String(), "64") {
		t.Errorf("stderr should mention the 64 cap: %q", errb.String())
	}
}

func TestCmdShare_AtomicValidation_BadPath(t *testing.T) {
	o, _, errb := newTestOpts(t)
	good := writeTempFile(t, "hello")
	bad := filepath.Join(t.TempDir(), "does-not-exist.txt")
	code := cmdShare(context.Background(), []string{good, bad}, o)
	if code != ExitSourceBad {
		t.Errorf("code = %d, want ExitSourceBad (got %d)", code, code)
	}
	if !strings.Contains(errb.String(), "does-not-exist.txt") {
		t.Errorf("stderr missing bad path: %q", errb.String())
	}
}

// ---------------------------------------------------------------------------
// selectDisplayName unit tests
// ---------------------------------------------------------------------------

func TestSelectDisplayName_AsBeatsAnonWhenSingle(t *testing.T) {
	got := selectDisplayName("/abs/secret.pdf", "my-name", true, 1)
	if got != "my-name" {
		t.Errorf("got %q, want %q (--as must win over --anon when N==1)", got, "my-name")
	}
}

func TestSelectDisplayName_AnonWhenNoAs(t *testing.T) {
	got := selectDisplayName("/abs/secret.PDF", "", true, 3)
	// Expect lowercased .pdf extension; slug is random, so check shape.
	if !strings.HasSuffix(got, ".pdf") {
		t.Errorf("got %q, want suffix .pdf", got)
	}
	if got == "secret.PDF" || got == "secret.pdf" {
		t.Errorf("got %q, expected a random slug, not the original filename", got)
	}
}

func TestSelectDisplayName_BaseFallback(t *testing.T) {
	got := selectDisplayName("/abs/dir/report.pdf", "", false, 1)
	if got != "report.pdf" {
		t.Errorf("got %q, want %q", got, "report.pdf")
	}
}

func TestSelectDisplayName_AsIgnoredWhenMultiple(t *testing.T) {
	// The CLI layer rejects --as when N>1 before ever reaching selectDisplayName,
	// but confirm the pure function's policy: --as requires total == 1.
	got := selectDisplayName("/abs/report.pdf", "override", false, 3)
	if got != "report.pdf" {
		t.Errorf("got %q, want %q (--as should not apply when total > 1)", got, "report.pdf")
	}
}

func TestCmdShare_MultipleFiles_HappyPath(t *testing.T) {
	t.Skip("no bootstrap override available; happy-path multi-file behavior is covered by manual smoke test in PR plan Task 6")
}
