// internal/admin/info.go
package admin

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
)

// Tokens and IDs are always generated in lowercase (see internal/share/token.go).
// Uppercase input is rejected — we do not case-fold, because downstream lookup
// compares the raw string against the lowercase map key.

// tokenRe matches a 26-char lowercase base32 token (RFC 4648 a-z2-7, no padding).
var tokenRe = regexp.MustCompile(`^[a-z2-7]{26}$`)

// idRe matches an 8-char lowercase base32 id.
var idRe = regexp.MustCompile(`^[a-z2-7]{8}$`)

// pathRe captures the token from /t/<token> or /t/<token>/<name>.
var pathRe = regexp.MustCompile(`(?:^|/)t/([a-z2-7]{26})(?:/.*)?$`)

// ParseInfoQuery normalizes one of {full URL, path /t/<tok>/<name>, bare token, bare id}
// into either a token (26-char base32) or an id (8-char base32). Exactly one
// of the returned values is populated on success.
func ParseInfoQuery(raw string) (token, id string, err error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", "", errors.New("empty query")
	}

	// URL with scheme: parse, extract path.
	if u, perr := url.Parse(s); perr == nil && u.Scheme != "" && u.Host != "" {
		if m := pathRe.FindStringSubmatch(u.Path); m != nil {
			return m[1], "", nil
		}
		return "", "", errors.New("URL does not contain a /t/<token> path")
	}

	// Path form, with or without leading slash.
	if m := pathRe.FindStringSubmatch(s); m != nil {
		return m[1], "", nil
	}

	// Bare token.
	if tokenRe.MatchString(s) {
		return s, "", nil
	}

	// Bare id.
	if idRe.MatchString(s) {
		return "", s, nil
	}

	return "", "", errors.New("not a recognizable URL, path, token, or id")
}
