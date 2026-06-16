package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/thedavidweng/monarchmoney-cli/internal/audit"
)

func TestAuditCmdRegistered(t *testing.T) {
	found := false
	for _, cmd := range RootCmd.Commands() {
		if cmd.Name() == "audit" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("audit command not registered on RootCmd")
	}
}

func TestAuditCleanupSubcommandRegistered(t *testing.T) {
	found := false
	for _, cmd := range auditCmd.Commands() {
		if cmd.Name() == "cleanup" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("cleanup subcommand not registered on auditCmd")
	}
}

func TestAuditCleanupFlagExists(t *testing.T) {
	f := auditCleanupCmd.Flags().Lookup("older-than")
	if f == nil {
		t.Fatal("audit cleanup command missing --older-than flag")
	}
	if f.DefValue != "30" {
		t.Fatalf("--older-than default = %q, want 30", f.DefValue)
	}
}

func TestAuditCleanupInvalidDays(t *testing.T) {
	oldJSONMode := jsonMode
	oldProfile := profile
	oldExitFunc := exitFunc
	jsonMode = true
	profile = "default"
	exitFunc = func(int) {}
	defer func() {
		jsonMode = oldJSONMode
		profile = oldProfile
		exitFunc = oldExitFunc
	}()

	auditCleanupDays = 0

	out := captureStdout(t, func() {
		auditCleanupCmd.Run(auditCleanupCmd, nil)
	})

	var env struct {
		OK    bool `json:"ok"`
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(trimNewline(out)), &env); err != nil {
		t.Fatalf("json.Unmarshal() error = %v; output=%q", err, out)
	}
	if env.OK {
		t.Fatal("expected ok=false for zero days")
	}
	if env.Error.Code != "INVALID_ARGUMENTS" {
		t.Fatalf("error code = %q, want INVALID_ARGUMENTS", env.Error.Code)
	}
}

func TestAuditCleanupInvalidDaysPlainText(t *testing.T) {
	oldJSONMode := jsonMode
	oldExitFunc := exitFunc
	jsonMode = false
	var gotExitCode int
	exitFunc = func(code int) { gotExitCode = code }
	defer func() {
		jsonMode = oldJSONMode
		exitFunc = oldExitFunc
	}()

	auditCleanupDays = -5

	// In non-JSON mode, handleError writes to stderr, so we just verify
	// the exit code is correct (InvalidArguments = exit code 2).
	auditCleanupCmd.Run(auditCleanupCmd, nil)

	if gotExitCode != 2 {
		t.Fatalf("exit code = %d, want 2 (InvalidArguments)", gotExitCode)
	}
}

func TestAuditCleanupRemovesOldFiles(t *testing.T) {
	// Create a temp audit directory with some old log files.
	dir := t.TempDir()

	oldDate := time.Now().AddDate(0, 0, -60).Format("2006-01-02")
	oldFile := filepath.Join(dir, oldDate+".jsonl")
	if err := os.WriteFile(oldFile, []byte(`{"command":"test"}`), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	recentDate := time.Now().AddDate(0, 0, -5).Format("2006-01-02")
	recentFile := filepath.Join(dir, recentDate+".jsonl")
	if err := os.WriteFile(recentFile, []byte(`{"command":"test"}`), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Directly test audit.Logger.Cleanup against the temp directory.
	logger := &audit.Logger{Dir: dir}
	removed, err := logger.Cleanup(30)
	if err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}
	if removed != 1 {
		t.Fatalf("Cleanup() removed = %d, want 1", removed)
	}

	// Old file should be gone, recent file should remain.
	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Fatalf("old file still exists: %v", err)
	}
	if _, err := os.Stat(recentFile); err != nil {
		t.Fatalf("recent file should still exist: %v", err)
	}
}

func TestAuditCleanupNoDirectory(t *testing.T) {
	logger := &audit.Logger{Dir: filepath.Join(t.TempDir(), "nonexistent")}
	removed, err := logger.Cleanup(30)
	if err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}
	if removed != 0 {
		t.Fatalf("Cleanup() removed = %d, want 0", removed)
	}
}

func TestAuditCleanupSkipsNonJSONL(t *testing.T) {
	dir := t.TempDir()

	oldDate := time.Now().AddDate(0, 0, -60).Format("2006-01-02")
	// Write a .txt file that looks like it could be an old log.
	txtFile := filepath.Join(dir, oldDate+".txt")
	if err := os.WriteFile(txtFile, []byte("not a log"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	logger := &audit.Logger{Dir: dir}
	removed, err := logger.Cleanup(30)
	if err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}
	if removed != 0 {
		t.Fatalf("Cleanup() removed = %d, want 0 (non-jsonl file should be skipped)", removed)
	}

	// Verify the file still exists.
	if _, err := os.Stat(txtFile); err != nil {
		t.Fatalf("txt file should still exist: %v", err)
	}
}
