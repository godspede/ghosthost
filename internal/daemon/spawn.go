package daemon

import "os/exec"

// SpawnDetached starts `selfPath daemon [extraArgs...]` in a detached
// process so the CLI can exit while the daemon continues running.
// extraArgs are passed before the "daemon" subcommand so flags like
// --config propagate correctly.
func SpawnDetached(selfPath string, extraArgs ...string) error {
	return spawnDetached(selfPath, extraArgs...)
}

var spawnExec = exec.Command
