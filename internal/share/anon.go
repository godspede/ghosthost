// internal/share/anon.go
package share

import (
	"crypto/rand"
	"path/filepath"
	"strings"
)

// AnonDisplayName returns a 6-char lowercase-base32 slug, followed by the
// lowercased extension of srcPath (if any). Example: "secret.PDF" -> "k9vm3q.pdf".
// The source file's original name and path are never included.
func AnonDisplayName(srcPath string) string {
	var b [4]byte // 32 bits -> 7 base32 chars, we take 6
	if _, err := rand.Read(b[:]); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	slug := strings.ToLower(b32.EncodeToString(b[:]))[:6]
	ext := strings.ToLower(filepath.Ext(srcPath))
	return slug + ext
}
