package source

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Resolve returns the absolute, symlink-resolved path of a user-supplied
// file path, after rejecting reparse points, UNC paths, and device paths.
// The result is what the daemon should serve.
func Resolve(input string) (string, error) {
	if input == "" {
		return "", errors.New("path is empty")
	}
	if strings.HasPrefix(input, `\\?\`) || strings.HasPrefix(input, `\\.\`) {
		return "", fmt.Errorf("raw/device paths not supported: %q", input)
	}
	if strings.HasPrefix(input, `\\`) {
		return "", fmt.Errorf("UNC paths not supported: %q", input)
	}

	abs, err := filepath.Abs(input)
	if err != nil {
		return "", fmt.Errorf("abs: %w", err)
	}
	abs = filepath.Clean(abs)

	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", fmt.Errorf("eval symlinks: %w", err)
	}

	info, err := os.Lstat(resolved)
	if err != nil {
		return "", fmt.Errorf("lstat: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("not a regular file: %q", resolved)
	}

	if err := rejectReparseOnPath(resolved); err != nil {
		return "", err
	}
	return resolved, nil
}
