// internal/config/cloudsync.go
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DetectCloudSync returns a non-empty reason string if dataDir appears to
// live inside a cloud-sync root (OneDrive, Dropbox, Google Drive).
// Heuristic — used to warn, not to hard-fail.
func DetectCloudSync(dataDir string) string {
	abs, err := filepath.Abs(dataDir)
	if err != nil {
		return ""
	}
	abs = strings.ToLower(filepath.Clean(abs))

	for _, env := range []string{"OneDrive", "OneDriveConsumer", "OneDriveCommercial", "DROPBOX"} {
		if root := os.Getenv(env); root != "" {
			if strings.HasPrefix(abs, strings.ToLower(filepath.Clean(root))+string(filepath.Separator)) ||
				abs == strings.ToLower(filepath.Clean(root)) {
				return fmt.Sprintf("data_dir is inside %s (%s)", env, root)
			}
		}
	}
	for _, marker := range []string{"onedrive", "dropbox", "google drive"} {
		if strings.Contains(abs, string(filepath.Separator)+marker+string(filepath.Separator)) {
			return fmt.Sprintf("data_dir path contains %q (possible cloud-sync folder)", marker)
		}
	}
	return ""
}
