package cache

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

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

func TestNewStoreSetsPrivateFilePermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "monarch.sqlite")
	store, err := NewStore(path)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	// Windows uses ACLs, not Unix permission bits.
	if runtime.GOOS != "windows" {
		if got, want := info.Mode().Perm(), os.FileMode(0600); got != want {
			t.Fatalf("permissions = %v, want %v", got, want)
		}
	}
}

func TestSaveMethodsReturnDatabaseErrors(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "monarch.sqlite"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	sqlDB, err := store.db.DB()
	if err != nil {
		t.Fatalf("DB() error = %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	if err := store.SaveAccounts([]Account{{
		ID:        "acc_1",
		UpdatedAt: time.Date(2026, 5, 9, 0, 0, 0, 0, time.UTC),
	}}); err == nil {
		t.Fatal("SaveAccounts() error = nil, want database error")
	}
	if err := store.SaveTransactions([]Transaction{{
		ID:   "tx_1",
		Date: time.Date(2026, 5, 9, 0, 0, 0, 0, time.UTC),
	}}); err == nil {
		t.Fatal("SaveTransactions() error = nil, want database error")
	}
}
