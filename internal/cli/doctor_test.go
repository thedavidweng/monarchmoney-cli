package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDoctorCmdRegistered(t *testing.T) {
	found := false
	for _, cmd := range RootCmd.Commands() {
		if cmd.Name() == "doctor" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("doctor command not registered on RootCmd")
	}
}

func TestDoctorPlainText(t *testing.T) {
	out := captureStdout(t, func() {
		doctorCmd.Run(doctorCmd, nil)
	})

	if !strings.Contains(out, "Monarch Money CLI Doctor") {
		t.Fatalf("output = %q, want header", out)
	}
	if !strings.Contains(out, "Version:") {
		t.Fatalf("output = %q, want Version line", out)
	}
	if !strings.Contains(out, "OS/Arch:") {
		t.Fatalf("output = %q, want OS/Arch line", out)
	}
	if !strings.Contains(out, "Config Path:") {
		t.Fatalf("output = %q, want Config Path line", out)
	}
	if !strings.Contains(out, "Session Path:") {
		t.Fatalf("output = %q, want Session Path line", out)
	}
}

func TestDoctorJSON(t *testing.T) {
	oldJSONMode := jsonMode
	oldProfile := profile
	jsonMode = true
	profile = "default"
	defer func() {
		jsonMode = oldJSONMode
		profile = oldProfile
	}()

	out := captureStdout(t, func() {
		doctorCmd.Run(doctorCmd, nil)
	})

	var env struct {
		OK   bool `json:"ok"`
		Data struct {
			Version string `json:"version"`
			OS      string `json:"os"`
			Arch    string `json:"arch"`
			Config  struct {
				Path   string `json:"path"`
				Exists bool   `json:"exists"`
			} `json:"config"`
			Session struct {
				Path   string `json:"path"`
				Exists bool   `json:"exists"`
			} `json:"session"`
		} `json:"data"`
		Meta struct {
			Command string `json:"command"`
		} `json:"meta"`
	}
	if err := json.Unmarshal([]byte(trimNewline(out)), &env); err != nil {
		t.Fatalf("json.Unmarshal() error = %v; output=%q", err, out)
	}
	if !env.OK {
		t.Fatal("expected ok=true")
	}
	if env.Data.OS == "" {
		t.Fatal("expected OS to be populated")
	}
	if env.Data.Arch == "" {
		t.Fatal("expected Arch to be populated")
	}
	if env.Meta.Command != "doctor" {
		t.Fatalf("meta.command = %q, want doctor", env.Meta.Command)
	}
}

func TestDoctorConnectFlag(t *testing.T) {
	// Verify that --connect flag exists and is accepted.
	f := doctorCmd.Flags().Lookup("connect")
	if f == nil {
		t.Fatal("doctor command missing --connect flag")
	}
	if f.DefValue != "false" {
		t.Fatalf("--connect default = %q, want false", f.DefValue)
	}
}

func TestDoctorConnectNotShownWithoutFlag(t *testing.T) {
	out := captureStdout(t, func() {
		doctorCmd.Run(doctorCmd, nil)
	})

	// Without --connect, the "API Connected:" line should not appear.
	if strings.Contains(out, "API Connected:") {
		t.Fatalf("output should not contain 'API Connected:' without --connect flag; got %q", out)
	}
}
