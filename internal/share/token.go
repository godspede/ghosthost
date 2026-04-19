package share

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"strings"
)

var b32 = base32.StdEncoding.WithPadding(base32.NoPadding)

// NewToken returns a 128-bit token encoded as 26 lowercase base32 chars.
func NewToken() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return strings.ToLower(b32.EncodeToString(b[:]))
}

// NewID returns a 40-bit ID encoded as 8 lowercase base32 chars.
func NewID() string {
	var b [5]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return strings.ToLower(b32.EncodeToString(b[:]))
}

// Digest returns the SHA-256 of a token. Only digests hit disk; raw tokens
// stay in memory.
func Digest(tok string) [32]byte {
	return sha256.Sum256([]byte(tok))
}

// EqualDigest compares two digests in constant time.
func EqualDigest(a, b [32]byte) bool {
	return subtle.ConstantTimeCompare(a[:], b[:]) == 1
}
