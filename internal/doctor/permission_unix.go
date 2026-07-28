//go:build !windows

package doctor

import "os"

func checkFilePermission(info os.FileInfo) bool {
	return info.Mode().Perm() == 0o600
}
