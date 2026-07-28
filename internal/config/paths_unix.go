//go:build !windows

package config

import (
	"os"
	"runtime"
)

var defaultDir = func() string {
	home, _ := os.UserHomeDir()
	xdgState := os.Getenv("XDG_STATE_HOME")
	return defaultDirFor(runtime.GOOS, home, xdgState, dirExists)
}
