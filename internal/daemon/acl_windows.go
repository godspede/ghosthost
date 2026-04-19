// internal/daemon/acl_windows.go
//go:build windows

package daemon

import "os/exec"

// restrictACLOwnerOnly uses icacls to grant only the current user and
// SYSTEM access to the lockfile and removes inherited permissions. Best
// effort; failure is logged but non-fatal.
func restrictACLOwnerOnly(path string) error {
	return exec.Command("icacls", path, "/inheritance:r", "/grant:r", "%USERNAME%:F", "SYSTEM:F").Run()
}
