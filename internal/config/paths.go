package config

import (
	"os"
	"path/filepath"
)

func DefaultDir() string {
	return defaultDir()
}

func DefaultConfigPath() string {
	return filepath.Join(DefaultDir(), "config.yaml")
}

func DefaultSessionPath() string {
	return filepath.Join(DefaultDir(), "session.json")
}

func DefaultAuditDir() string {
	return filepath.Join(DefaultDir(), "audit")
}

func DefaultCacheDir() string {
	return filepath.Join(DefaultDir(), "cache")
}

func DefaultCachePath() string {
	return filepath.Join(DefaultCacheDir(), "monarch.sqlite")
}

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

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
