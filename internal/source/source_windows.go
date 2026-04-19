//go:build windows

package source

import (
	"fmt"
	"path/filepath"
	"syscall"
)

// rejectReparseOnPath walks ancestor directories and returns an error if
// any path component has the reparse-point attribute set.
func rejectReparseOnPath(abs string) error {
	current := abs
	for {
		attrs, err := getAttrs(current)
		if err != nil {
			return fmt.Errorf("stat %q: %w", current, err)
		}
		if attrs&syscall.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
			return fmt.Errorf("path contains reparse point: %q", current)
		}
		parent := filepath.Dir(current)
		if parent == current {
			return nil
		}
		current = parent
	}
}

func getAttrs(path string) (uint32, error) {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	return syscall.GetFileAttributes(p)
}
