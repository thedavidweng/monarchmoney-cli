package scripts

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestLiveDriftReport(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("workflow runs on Linux")
	}
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("workflow requires jq")
	}
	data, err := os.ReadFile("../.github/workflows/api-drift.yml")
	if err != nil {
		t.Fatal(err)
	}
	var workflow struct {
		Jobs map[string]struct {
			Steps []struct {
				Name string `yaml:"name"`
				Run  string `yaml:"run"`
			} `yaml:"steps"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		t.Fatal(err)
	}
	var script string
	for _, step := range workflow.Jobs["live-drift"].Steps {
		if step.Name == "Probe live Monarch endpoints" {
			script = step.Run
		}
	}
	if script == "" {
		t.Fatal("live probe script not found")
	}

	for _, tc := range []struct {
		name, token, mode string
		wantReport        bool
		wantFailure       bool
	}{
		{name: "missing token"},
		{name: "successful probe", token: "synthetic", wantReport: true},
		{name: "failed probe", token: "synthetic", mode: "fail", wantReport: true, wantFailure: true},
		{name: "no probe executed", token: "synthetic", mode: "no-test", wantReport: true, wantFailure: true},
		{name: "build failed", token: "synthetic", mode: "build-fail", wantReport: true, wantFailure: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			goStub := `#!/bin/sh
printf '%s\n' '{"Action":"output","Output":"SYNTHETIC_PRIVATE_MARKER"}'
printf '%s\n' 'SYNTHETIC_PRIVATE_MARKER' >&2
if [ "$MOCK_MODE" = build-fail ]; then exit 1; fi
if [ "$MOCK_MODE" = no-test ]; then exit 0; fi
if [ "$MOCK_MODE" = fail ]; then
  printf '%s\n' '{"Action":"fail","Test":"TestLiveEndpointAvailability/identity/GetIdentity"}'
  printf '%s\n' '{"Action":"fail","Test":"TestLiveEndpointAvailability"}'
  exit 1
fi
printf '%s\n' '{"Action":"pass","Test":"TestLiveEndpointAvailability/identity/GetIdentity"}'
printf '%s\n' '{"Action":"pass","Test":"TestLiveEndpointAvailability"}'
`
			if err := os.WriteFile(filepath.Join(dir, "go"), []byte(goStub), 0o755); err != nil {
				t.Fatal(err)
			}
			outputPath := filepath.Join(dir, "output")
			cmd := exec.Command("bash", "-e", "-o", "pipefail", "-c", script)
			cmd.Dir = dir
			cmd.Env = []string{
				"PATH=" + dir + string(os.PathListSeparator) + os.Getenv("PATH"),
				"HOME=" + dir,
				"RUNNER_TEMP=" + dir,
				"GITHUB_OUTPUT=" + outputPath,
				"MONARCH_LIVE_TOKEN=" + tc.token,
				"MOCK_MODE=" + tc.mode,
			}
			out, err := cmd.CombinedOutput()
			if (err != nil) != tc.wantFailure {
				t.Fatalf("probe error = %v; output: %s", err, out)
			}
			if strings.Contains(string(out), "SYNTHETIC_PRIVATE_MARKER") {
				t.Fatalf("probe output exposed private marker: %s", out)
			}
			report, reportErr := os.ReadFile(filepath.Join(dir, "drift-report.txt"))
			if (reportErr == nil) != tc.wantReport {
				t.Fatalf("report error = %v, want report = %v", reportErr, tc.wantReport)
			}
			if strings.Contains(string(report), "SYNTHETIC_PRIVATE_MARKER") {
				t.Fatal("report exposed private marker")
			}
			output, outputErr := os.ReadFile(outputPath)
			wantOutput := "attempted=true\n"
			if tc.mode == "fail" {
				wantOutput += "detected=true\n"
			}
			if tc.wantReport && (outputErr != nil || string(output) != wantOutput) {
				t.Fatalf("probe step output = %q, error = %v", output, outputErr)
			}
			if !tc.wantReport && !os.IsNotExist(outputErr) {
				t.Fatalf("skipped probe created step output: %q, error = %v", output, outputErr)
			}
			tempFiles, err := filepath.Glob(filepath.Join(dir, "drift-*.??????"))
			if err != nil || len(tempFiles) != 0 {
				t.Fatalf("temporary raw files left behind: %v, error = %v", tempFiles, err)
			}
			if tc.mode == "fail" && !strings.Contains(string(report), "TestLiveEndpointAvailability/identity/GetIdentity") {
				t.Fatalf("failed test missing from report: %s", report)
			}
			if tc.token == "" && !strings.Contains(string(out), "::warning::") {
				t.Fatalf("missing token was not reported: %s", out)
			}
		})
	}
}
