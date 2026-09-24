package cli

import (
	"encoding/json"
	"testing"
)

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
	auditCleanupCmd.Run(auditCleanupCmd, nil)

	if gotExitCode != 2 {
		t.Fatalf("exit code = %d, want 2 (InvalidArguments)", gotExitCode)
	}
}
