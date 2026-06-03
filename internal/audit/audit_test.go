package audit

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLogWritesDailyFile(t *testing.T) {
	dir := t.TempDir()
	logger := &Logger{Dir: dir}

	if err := logger.Log(&Record{Command: "test.command", Result: "success"}); err != nil {
		t.Fatalf("Log() error = %v", err)
	}

	filename := time.Now().UTC().Format("2006-01-02") + ".jsonl"
	path := filepath.Join(dir, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	if len(data) == 0 {
		t.Fatal("Log() wrote empty file")
	}
}

func TestCleanupRemovesOldFiles(t *testing.T) {
	dir := t.TempDir()
	logger := &Logger{Dir: dir}

	// Create old files.
	oldDates := []string{
		time.Now().UTC().AddDate(0, 0, -10).Format("2006-01-02"),
		time.Now().UTC().AddDate(0, 0, -5).Format("2006-01-02"),
	}
	for _, d := range oldDates {
		path := filepath.Join(dir, d+".jsonl")
		if err := os.WriteFile(path, []byte(`{"test":true}`), 0600); err != nil {
			t.Fatalf("WriteFile(%s) error = %v", path, err)
		}
	}

	// Create a recent file (should not be removed).
	recent := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	recentPath := filepath.Join(dir, recent+".jsonl")
	if err := os.WriteFile(recentPath, []byte(`{"test":true}`), 0600); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", recentPath, err)
	}

	// Create a non-jsonl file (should be ignored).
	junkPath := filepath.Join(dir, "README.txt")
	if err := os.WriteFile(junkPath, []byte("ignore me"), 0600); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", junkPath, err)
	}

	// Cleanup files older than 7 days.
	removed, err := logger.Cleanup(7)
	if err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}
	if removed != 1 {
		t.Fatalf("Cleanup() removed = %d, want 1", removed)
	}

	// Verify the 10-day file was removed, 5-day and recent survive.
	if _, err := os.Stat(filepath.Join(dir, oldDates[0]+".jsonl")); !os.IsNotExist(err) {
		t.Fatal("10-day file should have been removed")
	}
	if _, err := os.Stat(filepath.Join(dir, oldDates[1]+".jsonl")); err != nil {
		t.Fatalf("5-day file should still exist: %v", err)
	}
	if _, err := os.Stat(recentPath); err != nil {
		t.Fatalf("recent file should still exist: %v", err)
	}
	if _, err := os.Stat(junkPath); err != nil {
		t.Fatalf("junk file should still exist: %v", err)
	}
}

func TestCleanupReturnsZeroForMissingDir(t *testing.T) {
	dir := t.TempDir()
	logger := &Logger{Dir: filepath.Join(dir, "nonexistent")}

	removed, err := logger.Cleanup(30)
	if err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}
	if removed != 0 {
		t.Fatalf("Cleanup() removed = %d, want 0", removed)
	}
}
