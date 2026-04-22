package share

import (
	"regexp"
	"strings"
	"testing"
)

func TestAnonDisplayName_Shape(t *testing.T) {
	cases := []struct {
		src     string
		wantExt string
	}{
		{"secret.pdf", ".pdf"},
		{"C:\\a\\b\\IMG_0001.PNG", ".png"}, // ext lowercased
		{"/etc/hosts", ""},                  // no ext
		{"archive.tar.gz", ".gz"},           // last ext only, by design
		{"noext", ""},
		{".bashrc", ".bashrc"}, // filepath.Ext(".bashrc") returns ".bashrc" (Go's actual behavior)
	}
	slugRe := regexp.MustCompile(`^[a-z2-7]{6}$`)
	for _, c := range cases {
		got := AnonDisplayName(c.src)
		base := got
		if c.wantExt != "" {
			if !strings.HasSuffix(got, c.wantExt) {
				t.Errorf("AnonDisplayName(%q) = %q, want suffix %q", c.src, got, c.wantExt)
			}
			base = strings.TrimSuffix(got, c.wantExt)
		}
		if !slugRe.MatchString(base) {
			t.Errorf("AnonDisplayName(%q) = %q, slug %q does not match [a-z2-7]{6}", c.src, got, base)
		}
	}
}

func TestAnonDisplayName_Unique(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		s := AnonDisplayName("x.txt")
		if seen[s] {
			t.Fatalf("collision after %d iterations: %q", i, s)
		}
		seen[s] = true
	}
}

func TestAnonDisplayName_PassesSanitize(t *testing.T) {
	for i := 0; i < 20; i++ {
		n := AnonDisplayName("x.pdf")
		if _, err := SanitizeDisplayName(n); err != nil {
			t.Fatalf("AnonDisplayName %q failed sanitize: %v", n, err)
		}
	}
}
