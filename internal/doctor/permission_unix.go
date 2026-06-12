//go:build !windows

package doctor

import "os"

// checkFilePermission verifies that the session file has restricted permissions (0600).
func checkFilePermission(info os.FileInfo) bool {
	return info.Mode().Perm() == 0600
}
