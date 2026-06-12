//go:build windows

package doctor

import "os"

// checkFilePermission on Windows always returns true.
// File permissions are managed by NTFS ACLs, not POSIX mode bits.
// os.Chmod is a no-op on Windows, so checking mode bits would always be a false negative.
func checkFilePermission(_ os.FileInfo) bool {
	return true
}
