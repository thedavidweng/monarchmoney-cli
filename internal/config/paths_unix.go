//go:build !windows

package config

import (
	"os"
	"path/filepath"
	"runtime"
)

// defaultDir returns the full application directory.
// On Linux: $XDG_STATE_HOME/monarchmoney-cli (default ~/.local/state/monarchmoney-cli)
// with legacy fallback to ~/.monarchmoney-cli if it exists.
// On macOS: ~/.monarchmoney-cli
var defaultDir = func() string {
	home, _ := os.UserHomeDir()
	xdgState := os.Getenv("XDG_STATE_HOME")
	return defaultDirFor(runtime.GOOS, home, xdgState, dirExists)
}

// defaultDirFor is the testable core of defaultDir.
func defaultDirFor(goos, home, xdgStateHome string, exists func(string) bool) string {
	legacy := filepath.Join(home, ".monarchmoney-cli")
	if goos != "linux" {
		return legacy
	}
	if xdgStateHome != "" && filepath.IsAbs(xdgStateHome) {
		return filepath.Join(xdgStateHome, "monarchmoney-cli")
	}
	xdgDefault := filepath.Join(home, ".local", "state", "monarchmoney-cli")
	if exists(legacy) && !exists(xdgDefault) {
		return legacy
	}
	return xdgDefault
}

// dirExists returns true if path exists and is a directory.
func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
