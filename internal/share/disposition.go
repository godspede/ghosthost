package share

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

type DispositionMode int

const (
	Inline DispositionMode = iota
	Attachment
)

// ContentDispositionHeader builds an RFC 6266 Content-Disposition value.
// Uses the ext-value (filename*) form for non-ASCII characters. Expects
// name to have already passed SanitizeDisplayName.
func ContentDispositionHeader(name string, mode DispositionMode) (string, error) {
	if _, err := SanitizeDisplayName(name); err != nil {
		return "", err
	}
	disp := "inline"
	if mode == Attachment {
		disp = "attachment"
	}
	asciiFallback := asciiFilenameFallback(name)
	if strings.ContainsAny(asciiFallback, `"\`) {
		return "", errors.New("internal: fallback filename contains reserved chars")
	}
	encoded := pathEscapeRFC5987(name)
	return fmt.Sprintf(`%s; filename=%q; filename*=UTF-8''%s`, disp, asciiFallback, encoded), nil
}

func asciiFilenameFallback(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '.', r == '-', r == '_', r == ' ', r == '(', r == ')':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}

func pathEscapeRFC5987(s string) string {
	escaped := url.QueryEscape(s)
	return strings.ReplaceAll(escaped, "+", "%20")
}
