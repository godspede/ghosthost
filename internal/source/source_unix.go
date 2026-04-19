//go:build !windows

package source

// rejectReparseOnPath is a no-op on non-Windows platforms. EvalSymlinks
// handles symlinks there; ghosthost's primary target is Windows.
func rejectReparseOnPath(abs string) error { return nil }
