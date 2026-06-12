//go:build windows

package config

import "os"

// userConfigDir returns the base directory for application configuration.
// On Windows, this is %APPDATA% (e.g. C:\Users\<user>\AppData\Roaming).
var userConfigDir = func() string {
	dir, _ := os.UserConfigDir()
	return dir
}

// configSubDir is the directory name under userConfigDir for this application.
const configSubDir = "monarchmoney-cli"
