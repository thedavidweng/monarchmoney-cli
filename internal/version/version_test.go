package version

import (
	"testing"
)

func TestGetVersionDefault(t *testing.T) {
	if got := GetVersion(); got == "" {
		t.Fatal("GetVersion() = empty, want non-empty")
	}
}

func TestGetCommitDefault(t *testing.T) {
	if got := GetCommit(); got == "" {
		t.Fatal("GetCommit() = empty, want non-empty")
	}
}

func TestGetCommitTruncatesLongHash(t *testing.T) {
	original := Commit
	Commit = "abcdef1234567890abcdef1234567890abcdef12"
	defer func() { Commit = original }()
	if got := GetCommit(); got != "abcdef123456" {
		t.Fatalf("GetCommit() = %q, want %q", got, "abcdef123456")
	}
}

func TestGetDateDefault(t *testing.T) {
	if got := GetDate(); got == "" {
		t.Fatal("GetDate() = empty, want non-empty")
	}
}

func TestGetBuiltByDefault(t *testing.T) {
	if got := GetBuiltBy(); got == "" {
		t.Fatal("GetBuiltBy() = empty, want non-empty")
	}
}

func TestGetBuildInfoReturnsString(t *testing.T) {
	got := GetBuildInfo()
	if got == "" {
		t.Fatal("GetBuildInfo() = empty, want non-empty")
	}
}
