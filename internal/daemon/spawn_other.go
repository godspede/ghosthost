//go:build !windows

package daemon

func spawnDetached(selfPath string, extraArgs ...string) error {
	args := append(append([]string{}, extraArgs...), "daemon")
	cmd := spawnExec(selfPath, args...)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
