package config

import (
	"path/filepath"
	"testing"
)

func TestDefaultDir_JoinsConfigDirWithSubDir(t *testing.T) {
	original := userConfigDir
	defer func() { userConfigDir = original }()

	userConfigDir = func() string { return filepath.Join("testdata", "appdata") }

	got := DefaultDir()
	want := filepath.Join("testdata", "appdata", configSubDir)
	if got != want {
		t.Fatalf("DefaultDir() = %q, want %q", got, want)
	}
}

func TestDefaultConfigPath_UnderDefaultDir(t *testing.T) {
	original := userConfigDir
	defer func() { userConfigDir = original }()

	userConfigDir = func() string { return filepath.Join("testdata", "appdata") }

	got := DefaultConfigPath()
	want := filepath.Join("testdata", "appdata", configSubDir, "config.yaml")
	if got != want {
		t.Fatalf("DefaultConfigPath() = %q, want %q", got, want)
	}
}

func TestDefaultSessionPath_UnderDefaultDir(t *testing.T) {
	original := userConfigDir
	defer func() { userConfigDir = original }()

	userConfigDir = func() string { return filepath.Join("testdata", "appdata") }

	got := DefaultSessionPath()
	want := filepath.Join("testdata", "appdata", configSubDir, "session.json")
	if got != want {
		t.Fatalf("DefaultSessionPath() = %q, want %q", got, want)
	}
}

func TestDefaultCachePath_UnderDefaultDir(t *testing.T) {
	original := userConfigDir
	defer func() { userConfigDir = original }()

	userConfigDir = func() string { return filepath.Join("testdata", "appdata") }

	got := DefaultCachePath()
	want := filepath.Join("testdata", "appdata", configSubDir, "cache", "monarch.sqlite")
	if got != want {
		t.Fatalf("DefaultCachePath() = %q, want %q", got, want)
	}
}
