package share

import (
	"testing"
	"time"
)

func TestShare_Active(t *testing.T) {
	now := time.Unix(1000, 0)
	s := Share{ExpiresAt: now.Add(time.Hour)}
	if !s.Active(now) {
		t.Fatal("should be active")
	}
	s.Revoked = true
	if s.Active(now) {
		t.Fatal("revoked share must not be active")
	}
	s.Revoked = false
	if s.Active(now.Add(2 * time.Hour)) {
		t.Fatal("expired share must not be active")
	}
}
