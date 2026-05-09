package cache

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestNewStoreFailsWhenParentPathIsAFile(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocked")
	if err := os.WriteFile(blocker, []byte("x"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := NewStore(filepath.Join(blocker, "monarch.sqlite")); err == nil {
		t.Fatal("NewStore() error = nil, want failure")
	}
}

func TestNewStoreReturnsOpenError(t *testing.T) {
	original := openSQLite
	openSQLite = func(string) (*gorm.DB, error) {
		return nil, errors.New("open failed")
	}
	defer func() { openSQLite = original }()

	if _, err := NewStore(filepath.Join(t.TempDir(), "monarch.sqlite")); err == nil {
		t.Fatal("NewStore() error = nil, want failure")
	}
}

func TestNewStoreReturnsMigrateError(t *testing.T) {
	originalOpen := openSQLite
	originalMigrate := migrateStore
	openSQLite = func(path string) (*gorm.DB, error) {
		return gorm.Open(sqlite.Open(path), &gorm.Config{})
	}
	migrateStore = func(*gorm.DB) error {
		return errors.New("migrate failed")
	}
	defer func() {
		openSQLite = originalOpen
		migrateStore = originalMigrate
	}()

	if _, err := NewStore(filepath.Join(t.TempDir(), "monarch.sqlite")); err == nil {
		t.Fatal("NewStore() error = nil, want failure")
	}
}
