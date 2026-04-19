// Package version exposes the ghosthost build version. Populated via
// -ldflags "-X github.com/godspede/ghosthost/internal/version.Version=..."
// at release time; "dev" otherwise.
package version

var (
	Version = "dev"
	Commit  = "unknown"
)
