package share

import "time"

// Share is an active (or historical) file share.
type Share struct {
	ID          string   // 8-char base32, surfaced to user
	Token       string   // 26-char base32, in URL path
	TokenDigest [32]byte // SHA-256 of Token, persisted to history
	SrcPath     string   // absolute, symlink-resolved path
	DisplayName string   // sanitized
	CreatedAt   time.Time
	ExpiresAt   time.Time
	Revoked     bool
}

// Active reports whether the share is currently accessible at time t.
func (s Share) Active(t time.Time) bool {
	return !s.Revoked && t.Before(s.ExpiresAt)
}
