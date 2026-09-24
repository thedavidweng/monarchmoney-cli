package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
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

// ─── Command discovery from help output ───

// Cobra help output lists commands under "Available Commands:" with 2-space indent.
// Example: "  accounts     Manage accounts"
// We only want to capture lines before "Flags:" section to avoid matching prose.
var helpCmdPattern = regexp.MustCompile(`^ {2}([a-z][a-z0-9-]*) +\S`)

func discoverCommands(t *testing.T, bin string) []string {
	t.Helper()
	stdout, _ := run(t, bin, "--help")
	var cmds []string
	inFlags := false
	for _, line := range strings.Split(stdout, "\n") {
		if strings.HasPrefix(line, "Flags:") {
			inFlags = true
		}
		if inFlags {
			continue
		}
		if m := helpCmdPattern.FindStringSubmatch(line); m != nil {
			cmd := m[1]
			if cmd == "help" || cmd == "completion" || cmd == "monarch" {
				continue
			}
			cmds = append(cmds, cmd)
		}
	}
	sort.Strings(cmds)
	return cmds
}

// ─── The golden list of expected commands ───
//
// AGENT INSTRUCTION: When you add a new command to monarch, add it to
// this list. TestAllCommandsInHelp fails until the list matches --help.

var requiredCommands = []string{
	"accounts", "analyze", "audit", "auth", "budgets",
	"cache", "cashflow", "categories", "credit",
	"debt", "doctor", "goals", "hledger", "household", "institutions", "investments",
	"merchants", "networth", "overview", "receipts", "recurring", "reports", "rules", "subscription",
	"tags", "transactions", "version", "whoami",
}

// ─── Meta tests ───

// TestAllCommandsInHelp verifies the CLI --help contains exactly the commands
// in requiredCommands. Adding a new command without updating the list fails.
func TestAllCommandsInHelp(t *testing.T) {
	bin := buildBinary(t)
	discovered := discoverCommands(t, bin)

	for _, required := range requiredCommands {
		found := false
		for _, d := range discovered {
			if d == required {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("required command %q not found in help. Add the command to Cobra OR remove it from requiredCommands.", required)
		}
	}

	for _, d := range discovered {
		known := false
		for _, required := range requiredCommands {
			if d == required {
				known = true
				break
			}
		}
		if !known {
			t.Errorf("command %q appears in help but is NOT in requiredCommands. Add it to the list (and add a test below).", d)
		}
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

func TestBinary_CommandList_Artifact(t *testing.T) {
	bin := buildBinary(t)
	first := discoverCommands(t, bin)
	if len(first) != len(requiredCommands) {
		t.Fatalf("commands = %d, want %d", len(first), len(requiredCommands))
	}
	content, _ := json.MarshalIndent(first, "", "  ")
	path := writeArtifact(t, "commands.json", string(content)+"\n")
	second := discoverCommands(t, bin)
	if len(second) != len(first) {
		t.Fatalf("command list not repeatable: %v vs %v", first, second)
	}
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("command list not repeatable: %v vs %v", first, second)
		}
	}
	t.Logf("command list artifact: %s (%d commands)", path, len(first))
}
