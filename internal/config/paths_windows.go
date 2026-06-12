//go:build windows

package config

import "os"

// defaultDir returns the full application directory.
// On Windows: %APPDATA%\monarchmoney-cli
var defaultDir = func() string {
	dir, _ := os.UserConfigDir()
	return dir + `\monarchmoney-cli`
}
