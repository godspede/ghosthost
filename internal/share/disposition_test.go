package share

import (
	"mime"
	"strings"
	"testing"
)

func TestContentDispositionHeader(t *testing.T) {
	cases := []struct {
		name string
		mode DispositionMode
		want string
	}{
		{"clip.mp4", Inline, `inline; filename="clip.mp4"; filename*=UTF-8''clip.mp4`},
		{"a b.mp4", Inline, `filename*=UTF-8''a%20b.mp4`},
		{"naïve.mp4", Attachment, `attachment;`},
		{`"quoted".mp4`, Inline, `filename*=UTF-8''%22quoted%22.mp4`},
	}
	for _, c := range cases {
		got, err := ContentDispositionHeader(c.name, c.mode)
		if err != nil {
			t.Fatalf("unexpected err for %q: %v", c.name, err)
		}
		if !strings.Contains(got, c.want) {
			t.Errorf("ContentDispositionHeader(%q) = %q, missing %q", c.name, got, c.want)
		}
		if _, _, err := mime.ParseMediaType(got); err != nil {
			t.Errorf("ContentDispositionHeader(%q) = %q, does not parse: %v", c.name, got, err)
		}
	}
}

func FuzzContentDispositionHeader(f *testing.F) {
	f.Add("clip.mp4")
	f.Add("a b.mp4")
	f.Add("naïve.mp4")
	f.Add("x\x00y")
	f.Fuzz(func(t *testing.T, name string) {
		if _, err := SanitizeDisplayName(name); err != nil {
			return
		}
		hdr, err := ContentDispositionHeader(name, Inline)
		if err != nil {
			t.Fatalf("err for sanitized %q: %v", name, err)
		}
		if strings.ContainsAny(hdr, "\r\n") {
			t.Fatalf("header contains CR/LF: %q", hdr)
		}
		if _, _, err := mime.ParseMediaType(hdr); err != nil {
			t.Fatalf("does not parse: %q: %v", hdr, err)
		}
	})
}
