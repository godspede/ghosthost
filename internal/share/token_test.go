package share

import (
	"regexp"
	"testing"
)

var tokenRe = regexp.MustCompile(`^[a-z2-7]{26}$`)
var idRe = regexp.MustCompile(`^[a-z2-7]{8}$`)

func TestNewToken_FormatAndEntropy(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 1000; i++ {
		tok := NewToken()
		if !tokenRe.MatchString(tok) {
			t.Fatalf("bad token format: %q", tok)
		}
		if seen[tok] {
			t.Fatalf("duplicate token in 1000 iters: %q", tok)
		}
		seen[tok] = true
	}
}

func TestNewID_FormatAndEntropy(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 1000; i++ {
		id := NewID()
		if !idRe.MatchString(id) {
			t.Fatalf("bad id format: %q", id)
		}
		seen[id] = true
	}
	if len(seen) < 990 {
		t.Fatalf("too many id collisions in 1000: %d unique", len(seen))
	}
}

func TestDigestAndEqual(t *testing.T) {
	tok := NewToken()
	d1 := Digest(tok)
	d2 := Digest(tok)
	if !EqualDigest(d1, d2) {
		t.Fatal("EqualDigest failed on identical digests")
	}
	if EqualDigest(d1, Digest(NewToken())) {
		t.Fatal("EqualDigest returned true for different tokens")
	}
}
