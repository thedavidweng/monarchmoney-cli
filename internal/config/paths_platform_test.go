package config

import (
	"path/filepath"
	"testing"
)

// ─── defaultDir tests ───

func TestDefaultDir_ReturnsFullAppDir(t *testing.T) {
	original := defaultDir
	defer func() { defaultDir = original }()

	defaultDir = func() string { return filepath.Join("testdata", "monarchmoney-cli") }

	got := DefaultDir()
	want := filepath.Join("testdata", "monarchmoney-cli")
	if got != want {
		t.Fatalf("DefaultDir() = %q, want %q", got, want)
	}
}

func TestDefaultConfigPath_UnderDefaultDir(t *testing.T) {
	original := defaultDir
	defer func() { defaultDir = original }()

	defaultDir = func() string { return filepath.Join("testdata", "monarchmoney-cli") }

	got := DefaultConfigPath()
	want := filepath.Join("testdata", "monarchmoney-cli", "config.yaml")
	if got != want {
		t.Fatalf("DefaultConfigPath() = %q, want %q", got, want)
	}
}

func TestDefaultSessionPath_UnderDefaultDir(t *testing.T) {
	original := defaultDir
	defer func() { defaultDir = original }()

	defaultDir = func() string { return filepath.Join("testdata", "monarchmoney-cli") }

	got := DefaultSessionPath()
	want := filepath.Join("testdata", "monarchmoney-cli", "session.json")
	if got != want {
		t.Fatalf("DefaultSessionPath() = %q, want %q", got, want)
	}
}

func TestDefaultCachePath_UnderDefaultDir(t *testing.T) {
	original := defaultDir
	defer func() { defaultDir = original }()

	defaultDir = func() string { return filepath.Join("testdata", "monarchmoney-cli") }

	got := DefaultCachePath()
	want := filepath.Join("testdata", "monarchmoney-cli", "cache", "monarch.sqlite")
	if got != want {
		t.Fatalf("DefaultCachePath() = %q, want %q", got, want)
	}
}
