//go:build windows

package config

import (
	"os"
	"path/filepath"
)

// defaultDir returns the full application directory.
// On Windows: %APPDATA%\monarchmoney-cli
var defaultDir = func() string {
	dir, _ := os.UserConfigDir()
	return filepath.Join(dir, "monarchmoney-cli")
}
