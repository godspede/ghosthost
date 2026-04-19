package server

import (
	"strings"
	"testing"
)

func TestContentTypeFor(t *testing.T) {
	cases := []struct {
		name   string
		prefix string // prefix match (ignores charset params, etc.)
	}{
		// Video
		{"a.mp4", "video/mp4"},
		{"a.mkv", "video/x-matroska"},
		{"a.webm", "video/webm"},
		{"a.mov", "video/quicktime"},
		{"a.avi", "video/x-msvideo"},
		// Audio
		{"a.mp3", "audio/mpeg"},
		{"a.flac", "audio/flac"},
		{"a.opus", "audio/ogg"},
		{"a.m4a", "audio/mp4"},
		// Image
		{"a.png", "image/png"},
		{"a.jpg", "image/jpeg"},
		{"a.svg", "image/svg+xml"},
		{"a.avif", "image/avif"},
		// Text/docs
		{"a.txt", "text/plain"},
		{"a.json", "application/json"},
		{"a.pdf", "application/pdf"},
		{"a.yaml", "text/yaml"},
	}
	for _, c := range cases {
		got := contentTypeFor(c.name)
		if !strings.HasPrefix(got, c.prefix) {
			t.Errorf("contentTypeFor(%q) = %q; want prefix %q", c.name, got, c.prefix)
		}
	}
}

func TestContentTypeFor_Miss(t *testing.T) {
	if got := contentTypeFor("weirdo.xyz"); got != "" {
		t.Errorf("contentTypeFor(.xyz) = %q; want empty", got)
	}
	if got := contentTypeFor("noext"); got != "" {
		t.Errorf("contentTypeFor(no-ext) = %q; want empty", got)
	}
}

func TestContentTypeFor_CaseInsensitive(t *testing.T) {
	if got := contentTypeFor("CLIP.MP4"); !strings.HasPrefix(got, "video/mp4") {
		t.Errorf("case-insensitive .MP4 failed, got %q", got)
	}
}
