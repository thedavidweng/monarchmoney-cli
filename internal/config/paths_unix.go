//go:build !windows

package config

import (
	"os"
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
