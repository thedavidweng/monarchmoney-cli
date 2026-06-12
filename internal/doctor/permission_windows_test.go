//go:build windows

package doctor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckFilePermission_WindowsAlwaysTrue(t *testing.T) {
	// On Windows, file permissions are managed by ACLs not POSIX mode bits.
	// checkFilePermission should always return true regardless of mode.
	path := filepath.Join(t.TempDir(), "testfile")
	if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	if !checkFilePermission(info) {
		t.Fatal("on Windows, checkFilePermission should always return true")
	}
}
