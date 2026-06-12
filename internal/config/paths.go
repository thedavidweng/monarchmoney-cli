package config

import (
	"path/filepath"
)

// DefaultDir returns the default configuration directory.
// On Unix this is ~/.monarchmoney-cli; on Windows it is %APPDATA%\monarchmoney-cli.
func DefaultDir() string {
	return filepath.Join(userConfigDir(), configSubDir)
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
