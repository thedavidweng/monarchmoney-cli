package config

import (
	"os"
	"path/filepath"
)

// DefaultDir returns the default configuration directory.
// On Linux:   $XDG_STATE_HOME/monarchmoney-cli (default ~/.local/state/monarchmoney-cli)
// On macOS:   ~/.monarchmoney-cli
// On Windows: %APPDATA%\monarchmoney-cli
func DefaultDir() string {
	return defaultDir()
}

// DefaultConfigPath returns the default configuration file path.
func DefaultConfigPath() string {
	return filepath.Join(DefaultDir(), "config.yaml")
}

// DefaultSessionPath returns the default session file path.
func DefaultSessionPath() string {
	return filepath.Join(DefaultDir(), "session.json")
}

// DefaultAuditDir returns the default audit log directory.
func DefaultAuditDir() string {
	return filepath.Join(DefaultDir(), "audit")
}

// DefaultCacheDir returns the default cache directory.
func DefaultCacheDir() string {
	return filepath.Join(DefaultDir(), "cache")
}

// DefaultCachePath returns the default cache file path.
func DefaultCachePath() string {
	return filepath.Join(DefaultCacheDir(), "monarch.sqlite")
}

// defaultDirFor is the testable core of defaultDir on Unix.
// On non-Linux: always <home>/.monarchmoney-cli.
// On Linux: $XDG_STATE_HOME/monarchmoney-cli with legacy fallback.
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
