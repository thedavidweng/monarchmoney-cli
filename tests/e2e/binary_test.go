package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
		cachedBin = filepath.Join(dir, "monarch")
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
	for dir := pwd; dir != "/" && dir != "."; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
	}
	return filepath.Join(os.Getenv("HOME"), "Development", "monarchmoney-cli"), nil
}

// ─── Execution helper ───

func run(t *testing.T, bin string, args ...string) (stdout string, exitCode int) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Env = []string{
		"HOME=" + t.TempDir(),
		"PATH=/usr/bin:/bin:/usr/local/bin",
		"TERM=dumb",
		"NO_COLOR=1",
	}
	outBytes, err := cmd.CombinedOutput()
	stdout = string(outBytes)
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	} else if err != nil {
		t.Fatalf("unexpected exec error: %v", err)
	}
	recordCommand(args)
	return stdout, exitCode
}

func assertValidEnvelope(t *testing.T, stdout string, wantCommand string) map[string]any {
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
	if meta, ok := envelope["meta"].(map[string]any); ok {
		if got, ok := meta["command"].(string); ok && wantCommand != "" && got != wantCommand {
			t.Fatalf("command = %q, want %q", got, wantCommand)
		}
		if _, ok := meta["schema_version"]; !ok {
			t.Fatalf("meta missing 'schema_version': %+v", meta)
		}
	} else {
		t.Fatalf("envelope.meta is not an object: %+v", envelope)
	}
	return envelope
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
var helpCmdPattern = regexp.MustCompile(`^  ([a-z][a-z0-9-]*) +\S`)

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
// AGENT INSTRUCTION: When you add a new command to monarch:
// 1. Add it to this list
// 2. Add a test below that exercises it
// 3. Run go test -v ./tests/e2e/... to verify

var requiredCommands = []string{
	"accounts", "analyze", "audit", "auth", "budgets",
	"cache", "cashflow", "categories", "credit",
	"doctor", "goals", "institutions", "investments",
	"networth", "recurring", "rules", "subscription",
	"tags", "transactions", "version",
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

// ─── Individual command tests ───
//
// AGENT INSTRUCTION:
// When you add a new command to monarch, add a test here that runs it.
// The test name must be TestBinary_<CommandName>_* so it is discoverable.

func TestBinary_Accounts_Help(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "accounts", "--help")
	requireZero(t, code, stdout)
	for _, sub := range []string{"list", "show", "types", "holdings", "history", "refresh", "update", "delete"} {
		if !strings.Contains(stdout, sub) {
			t.Errorf("accounts help missing subcommand %q", sub)
		}
	}
}

func TestBinary_Analyze_Help(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "analyze", "--help")
	requireZero(t, code, stdout)
	if !strings.Contains(stdout, "Usage:") {
		t.Fatal("analyze help missing Usage:")
	}
}

func TestBinary_Audit_Help(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "audit", "--help")
	requireZero(t, code, stdout)
	for _, sub := range []string{"cleanup"} {
		if !strings.Contains(stdout, sub) {
			t.Fatalf("audit help missing subcommand %q", sub)
		}
	}
}

func TestBinary_Auth_Help(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "auth", "--help")
	requireZero(t, code, stdout)
	for _, sub := range []string{"login", "logout", "status", "session"} {
		if !strings.Contains(stdout, sub) {
			t.Errorf("auth help missing subcommand %q", sub)
		}
	}
}

func TestBinary_Budgets_Help(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "budgets", "--help")
	requireZero(t, code, stdout)
}

func TestBinary_Cache_Help(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "cache", "--help")
	requireZero(t, code, stdout)
}

func TestBinary_Cashflow_Help(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "cashflow", "--help")
	requireZero(t, code, stdout)
	for _, flag := range []string{"--from", "--to"} {
		if !strings.Contains(stdout, flag) {
			t.Errorf("cashflow help missing flag %q", flag)
		}
	}
	for _, sub := range []string{"categories", "list", "merchants", "spending", "summary", "trends"} {
		if !strings.Contains(stdout, sub) {
			t.Errorf("cashflow help missing subcommand %q", sub)
		}
	}
}

func TestBinary_Categories_Help(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "categories", "--help")
	requireZero(t, code, stdout)
	if !strings.Contains(stdout, "list") {
		t.Errorf("categories help missing 'list'")
	}
}

func TestBinary_Credit_Help(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "credit", "--help")
	requireZero(t, code, stdout)
}

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

func TestBinary_Goals_Help(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "goals", "--help")
	requireZero(t, code, stdout)
}

func TestBinary_Institutions_Help(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "institutions", "--help")
	requireZero(t, code, stdout)
}

func TestBinary_Investments_Help(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "investments", "--help")
	requireZero(t, code, stdout)
	for _, sub := range []string{"portfolio", "performance"} {
		if !strings.Contains(stdout, sub) {
			t.Errorf("investments help missing subcommand %q", sub)
		}
	}
}

func TestBinary_Networth_Help(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "networth", "--help")
	requireZero(t, code, stdout)
}

func TestBinary_Recurring_Help(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "recurring", "--help")
	requireZero(t, code, stdout)
	if !strings.Contains(stdout, "list") {
		t.Errorf("recurring help missing 'list'")
	}
}

func TestBinary_Rules_Help(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "rules", "--help")
	requireZero(t, code, stdout)
}

func TestBinary_Subscription_Help(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "subscription", "--help")
	requireZero(t, code, stdout)
	if !strings.Contains(stdout, "show") {
		t.Errorf("subscription help missing 'show'")
	}
}

func TestBinary_Tags_Help(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "tags", "--help")
	requireZero(t, code, stdout)
	if !strings.Contains(stdout, "list") {
		t.Errorf("tags help missing 'list'")
	}
}

func TestBinary_Transactions_Help(t *testing.T) {
	bin := buildBinary(t)
	stdout, code := run(t, bin, "transactions", "--help")
	requireZero(t, code, stdout)
	if !strings.Contains(stdout, "list") {
		t.Errorf("transactions help missing 'list'")
	}
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

// ─── Coverage check ───

var (
	executedCmds = make(map[string]bool)
	mu           sync.Mutex
)

func recordCommand(args []string) {
	if len(args) == 0 {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	var path string
	if len(args) >= 2 && !strings.HasPrefix(args[1], "-") {
		path = args[0] + " " + args[1]
	} else {
		path = args[0]
	}
	executedCmds[path] = true
}

func TestCoverageReport(t *testing.T) {
	var uncovered []string
	for _, cmd := range requiredCommands {
		mu.Lock()
		covered := executedCmds[cmd]
		for executed := range executedCmds {
			if strings.HasPrefix(executed, cmd+" ") {
				covered = true
				break
			}
		}
		mu.Unlock()
		if !covered {
			uncovered = append(uncovered, cmd)
		}
	}

	if len(uncovered) > 0 {
		t.Errorf("%d commands have NO E2E test coverage: %v. Add a TestBinary_<command>_Help test above this one.", len(uncovered), uncovered)
	}
	t.Logf("E2E command coverage: %d/%d commands tested", len(requiredCommands)-len(uncovered), len(requiredCommands))
}
