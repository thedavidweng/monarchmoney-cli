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

// ─── defaultDirFor tests (Linux XDG logic) ───

func TestDefaultDirFor_NonLinux_UsesHome(t *testing.T) {
	got := defaultDirFor("darwin", "/Users/alice", "/Users/alice/.local/state", func(string) bool { return false })
	want := filepath.Join("/Users/alice", ".monarchmoney-cli")
	if got != want {
		t.Fatalf("defaultDirFor(darwin) = %q, want %q", got, want)
	}
}

func TestDefaultDirFor_Linux_UsesXDGStateHome(t *testing.T) {
	got := defaultDirFor("linux", "/home/alice", "/custom/state", func(string) bool { return false })
	want := filepath.Join("/custom/state", "monarchmoney-cli")
	if got != want {
		t.Fatalf("defaultDirFor(linux, xdg=/custom/state) = %q, want %q", got, want)
	}
}

func TestDefaultDirFor_Linux_DefaultStateDir(t *testing.T) {
	got := defaultDirFor("linux", "/home/alice", "", func(string) bool { return false })
	want := filepath.Join("/home/alice", ".local", "state", "monarchmoney-cli")
	if got != want {
		t.Fatalf("defaultDirFor(linux, xdg=) = %q, want %q", got, want)
	}
}

func TestDefaultDirFor_Linux_LegacyFallback(t *testing.T) {
	// Legacy exists, XDG default does not → use legacy.
	exists := func(p string) bool {
		return p == filepath.Join("/home/alice", ".monarchmoney-cli")
	}
	got := defaultDirFor("linux", "/home/alice", "", exists)
	want := filepath.Join("/home/alice", ".monarchmoney-cli")
	if got != want {
		t.Fatalf("defaultDirFor(linux, legacy exists) = %q, want %q", got, want)
	}
}

func TestDefaultDirFor_Linux_XDGTakesPrecedence(t *testing.T) {
	// Both legacy and XDG exist → use XDG.
	exists := func(string) bool { return true }
	got := defaultDirFor("linux", "/home/alice", "", exists)
	want := filepath.Join("/home/alice", ".local", "state", "monarchmoney-cli")
	if got != want {
		t.Fatalf("defaultDirFor(linux, both exist) = %q, want %q", got, want)
	}
}

func TestDefaultDirFor_Linux_RelativeXDG_Ignored(t *testing.T) {
	got := defaultDirFor("linux", "/home/alice", "relative/path", func(string) bool { return false })
	want := filepath.Join("/home/alice", ".local", "state", "monarchmoney-cli")
	if got != want {
		t.Fatalf("defaultDirFor(linux, relative xdg) = %q, want %q", got, want)
	}
}
