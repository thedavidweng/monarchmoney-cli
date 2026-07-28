//go:build windows

package config

import (
	"os"
	"path/filepath"
)

var defaultDir = func() string {
	dir, _ := os.UserConfigDir()
	return filepath.Join(dir, "monarchmoney-cli")
}
