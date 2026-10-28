package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

// ─── Build cache ───

var (
	buildOnce sync.Once
	cachedBin string
	buildErr  error
)

func buildBinary(t *testing.T) string {
	t.Helper()
	buildOnce.Do(func() {
		projectRoot, err := findProjectRoot()
		if err != nil {
			buildErr = err
			return
		}
		dir, err := os.MkdirTemp("", "monarch-e2e-*")
		if err != nil {
			buildErr = err
			return
		}
		binName := "monarch"
		if runtime.GOOS == "windows" {
			binName += ".exe"
		}
		cachedBin = filepath.Join(dir, binName)
		cmd := exec.Command("go", "build", "-o", cachedBin, "./cmd/monarch")
		cmd.Dir = projectRoot
		if out, err := cmd.CombinedOutput(); err != nil {
			buildErr = fmt.Errorf("build failed: %v\n%s", err, out)
			return
		}
	})
	if buildErr != nil {
		t.Fatalf("binary build failed: %v", buildErr)
	}
	return cachedBin
}

func findProjectRoot() (string, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for dir := pwd; ; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		next := filepath.Dir(dir)
		if next == dir {
			break
		}
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Development", "monarchmoney-cli"), nil
}

// ─── findProjectRoot helper ───

// ─── Execution helper ───

func run(t *testing.T, bin string, args ...string) (stdout string, exitCode int) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	tmpHome := t.TempDir()
	cmd.Env = []string{
		"HOME=" + tmpHome,
		"USERPROFILE=" + tmpHome,
		"PATH=" + os.Getenv("PATH"),
		"TERM=dumb",
		"NO_COLOR=1",
	}
	// Windows needs SYSTEMROOT for DLL lookup and COMSPEC for cmd.exe.
	if runtime.GOOS == "windows" {
		cmd.Env = append(cmd.Env,
			"SYSTEMROOT="+os.Getenv("SYSTEMROOT"),
			"COMSPEC="+os.Getenv("COMSPEC"),
		)
	}
	outBytes, err := cmd.CombinedOutput()
	stdout = string(outBytes)
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	} else if err != nil {
		t.Fatalf("unexpected exec error: %v", err)
	}
	return stdout, exitCode
}

func assertValidEnvelope(t *testing.T, stdout, wantCommand string) {
	t.Helper()
	var envelope map[string]any
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Fatalf("invalid JSON envelope: %v\noutput: %s", err, stdout)
	}
	if envelope["ok"] != true {
		t.Fatalf("envelope.ok = false: %+v", envelope)
	}
	if _, ok := envelope["data"]; !ok {
		t.Fatalf("envelope missing 'data': %+v", envelope)
	}
	meta, ok := envelope["meta"].(map[string]any)
	if !ok {
		t.Fatalf("envelope.meta is not an object: %+v", envelope)
	}
	if got, ok := meta["command"].(string); ok && wantCommand != "" && got != wantCommand {
		t.Fatalf("command = %q, want %q", got, wantCommand)
	}
	if _, ok := meta["schema_version"]; !ok {
		t.Fatalf("meta missing 'schema_version': %+v", meta)
	}
}

func requireZero(t *testing.T, code int, stdout string) {
	t.Helper()
	if code != 0 {
		t.Fatalf("expected exit 0, got %d. output:\n%s", code, stdout)
	}
}

// ─── Offline command tests ───

func TestBinary_Doctor(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "doctor")
	requireZero(t, code, stdout)
	if !strings.Contains(stdout, "Monarch Money CLI Doctor") {
		t.Fatal("doctor output missing header")
	}
}

func TestBinary_Doctor_JSON(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "doctor", "--json")
	requireZero(t, code, stdout)
	assertValidEnvelope(t, stdout, "doctor")
}

func TestBinary_Version(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "--version")
	requireZero(t, code, stdout)
	if !strings.Contains(stdout, "monarch version") {
		t.Fatalf("version output missing banner: %q", stdout)
	}
}

func TestBinary_Version_JSON(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "version", "--json")
	requireZero(t, code, stdout)
	assertValidEnvelope(t, stdout, "version")
}

// ─── Edge cases ───

func TestBinary_UnknownCommand(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "nonexistent")
	if code == 0 {
		t.Fatalf("expected non-zero exit, got 0. output: %s", stdout)
	}
}

func TestBinary_EmptyArgs_ShowsHelp(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin)
	requireZero(t, code, stdout)
	if !strings.Contains(stdout, "Usage:") {
		t.Fatalf("no args should show help, got: %q", stdout)
	}
}

func TestBinary_GlobalFlags_JSON(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "doctor", "--json")
	requireZero(t, code, stdout)
	assertValidEnvelope(t, stdout, "doctor")
}

func TestBinary_GlobalFlags_Pretty(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "version", "--json", "--pretty")
	requireZero(t, code, stdout)
	if !strings.Contains(stdout, "\n  ") {
		t.Fatalf("--pretty missing indentation")
	}
}

func writeArtifact(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", name, err)
	}
	return path
}

func envelopeWithoutDuration(t *testing.T, stdout string) map[string]any {
	t.Helper()
	var envelope map[string]any
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Fatalf("invalid JSON envelope: %v\noutput: %s", err, stdout)
	}
	if meta, ok := envelope["meta"].(map[string]any); ok {
		delete(meta, "duration_ms")
		delete(meta, "request_id")
	}
	return envelope
}

func TestBinary_Version_JSON_Artifact(t *testing.T) {
	bin := buildBinary(t)
	first, code := run(t, bin, "version", "--json")
	requireZero(t, code, first)
	assertValidEnvelope(t, first, "version")
	firstPath := writeArtifact(t, "version.json", first)
	second, code := run(t, bin, "version", "--json")
	requireZero(t, code, second)
	assertValidEnvelope(t, second, "version")
	firstEnv, secondEnv := envelopeWithoutDuration(t, first), envelopeWithoutDuration(t, second)
	firstJSON, _ := json.Marshal(firstEnv)
	secondJSON, _ := json.Marshal(secondEnv)
	if !bytes.Equal(firstJSON, secondJSON) {
		t.Fatalf("version artifact not repeatable:\nfirst: %s\nsecond: %s", firstJSON, secondJSON)
	}
	t.Logf("version artifact: %s", firstPath)
}

func TestBinary_Doctor_JSON_Artifact(t *testing.T) {
	bin := buildBinary(t)
	first, code := run(t, bin, "doctor", "--json")
	requireZero(t, code, first)
	assertValidEnvelope(t, first, "doctor")
	firstPath := writeArtifact(t, "doctor.json", first)
	second, code := run(t, bin, "doctor", "--json")
	requireZero(t, code, second)
	assertValidEnvelope(t, second, "doctor")
	firstEnv, secondEnv := envelopeWithoutDuration(t, first), envelopeWithoutDuration(t, second)
	stable := func(env map[string]any) map[string]any {
		data, _ := env["data"].(map[string]any)
		out := map[string]any{}
		for _, k := range []string{"os", "arch", "version"} {
			out[k] = data[k]
		}
		for _, k := range []string{"session", "config", "safety", "network"} {
			if sub, ok := data[k].(map[string]any); ok {
				out[k] = sub["exists"]
			}
		}
		return out
	}
	firstJSON, _ := json.Marshal(stable(firstEnv))
	secondJSON, _ := json.Marshal(stable(secondEnv))
	if !bytes.Equal(firstJSON, secondJSON) {
		t.Fatalf("doctor artifact not repeatable:\nfirst: %s\nsecond: %s", firstJSON, secondJSON)
	}
	t.Logf("doctor artifact: %s", firstPath)
}
