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
