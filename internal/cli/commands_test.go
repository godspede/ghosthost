package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/godspede/ghosthost/internal/admin"
)

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
