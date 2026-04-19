package share

import (
	"errors"
	"strings"
)

var reservedWindowsNames = map[string]bool{
	"CON": true, "PRN": true, "AUX": true, "NUL": true,
	"COM1": true, "COM2": true, "COM3": true, "COM4": true, "COM5": true,
	"COM6": true, "COM7": true, "COM8": true, "COM9": true,
	"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true, "LPT5": true,
	"LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true,
}

// SanitizeDisplayName validates a user-supplied display name for use in URL
// path and Content-Disposition. Rejects anything that could enable path
// traversal, header injection, or Windows-reserved devices.
func SanitizeDisplayName(name string) (string, error) {
	if name == "" {
		return "", errors.New("display name is empty")
	}
	if len(name) > 255 {
		return "", errors.New("display name exceeds 255 bytes")
	}
	if name == "." || name == ".." {
		return "", errors.New("display name is a path segment")
	}
	for _, r := range name {
		switch {
		case r == 0, r == '\n', r == '\r', r == '\t':
			return "", errors.New("display name contains control character")
		case r == '/', r == '\\':
			return "", errors.New("display name contains path separator")
		case r < 0x20, r == 0x7f:
			return "", errors.New("display name contains control character")
		}
	}
	base := name
	if dot := strings.LastIndexByte(base, '.'); dot > 0 {
		base = base[:dot]
	}
	if reservedWindowsNames[strings.ToUpper(base)] {
		return "", errors.New("display name is a Windows reserved device name")
	}
	return name, nil
}
