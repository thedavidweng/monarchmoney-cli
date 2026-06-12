//go:build !windows

package doctor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckFilePermission_0600(t *testing.T) {
	path := filepath.Join(t.TempDir(), "testfile")
	if err := os.WriteFile(path, []byte("secret"), 0600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	if !checkFilePermission(info) {
		t.Fatal("checkFilePermission(0600) = false, want true")
	}
}

func TestCheckFilePermission_0644(t *testing.T) {
	path := filepath.Join(t.TempDir(), "testfile")
	if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	if checkFilePermission(info) {
		t.Fatal("checkFilePermission(0644) = true, want false")
	}
}
