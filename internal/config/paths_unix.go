//go:build !windows

package config

import "os"

// userConfigDir returns the base directory for application configuration.
// On Unix, this is the user's home directory.
var userConfigDir = func() string {
	home, _ := os.UserHomeDir()
	return home
}

// configSubDir is the directory name under userConfigDir for this application.
const configSubDir = ".monarchmoney-cli"
