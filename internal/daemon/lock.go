// internal/daemon/lock.go
package daemon

// Lockfile represents the daemon's exclusive lock. The OS releases it on
// process exit, including crashes.
type Lockfile interface {
	// Read returns the advisory metadata written into the file.
	Read() (Meta, error)
	// Write atomically replaces the file's content with m and fsyncs.
	Write(m Meta) error
	// Close releases the lock and closes the underlying handle.
	Close() error
}

// Meta is the advisory content of the lockfile.
type Meta struct {
	PID       int    `json:"pid"`
	AdminPort int    `json:"admin_port"`
	Secret    string `json:"secret"`
	Version   string `json:"version"`
}

// AcquireLock obtains an exclusive lock on path. Returns ErrLocked if
// another process already holds it.
func AcquireLock(path string) (Lockfile, error) { return acquireLock(path) }
