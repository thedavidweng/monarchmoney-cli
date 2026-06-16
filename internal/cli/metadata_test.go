package cli

import (
	"encoding/json"
	"testing"
	"time"
)

func TestEnvelopeWithWarningsMultiple(t *testing.T) {
	originalProfile := profile
	profile = "test-prof"
	defer func() { profile = originalProfile }()

	warnings := []string{
		"uses legacy field",
		"deprecated endpoint",
	}
	env := envelopeWithWarnings("accounts.list", map[string]int{"count": 5}, time.Now(), warnings...)

	if len(env.Meta.Warnings) != 2 {
		t.Fatalf("warnings count = %d, want 2", len(env.Meta.Warnings))
	}
	if env.Meta.Warnings[0] != "uses legacy field" {
		t.Fatalf("warnings[0] = %q", env.Meta.Warnings[0])
	}
	if env.Meta.Warnings[1] != "deprecated endpoint" {
		t.Fatalf("warnings[1] = %q", env.Meta.Warnings[1])
	}
	if env.Meta.Command != "accounts.list" {
		t.Fatalf("command = %q, want accounts.list", env.Meta.Command)
	}
	if env.Meta.Profile != "test-prof" {
		t.Fatalf("profile = %q, want test-prof", env.Meta.Profile)
	}
}

func TestEnvelopeWithWarningsNone(t *testing.T) {
	originalProfile := profile
	profile = "default"
	defer func() { profile = originalProfile }()

	env := envelopeWithWarnings("test.cmd", nil, time.Now())
	if env.Meta.Warnings != nil {
		t.Fatalf("warnings = %v, want nil when no warnings provided", env.Meta.Warnings)
	}
	if !env.OK {
		t.Fatal("expected ok=true")
	}
}

func TestEnvelopeWithWarningsJSON(t *testing.T) {
	originalProfile := profile
	profile = "default"
	defer func() { profile = originalProfile }()

	env := envelopeWithWarnings("tx.search", map[string]string{"q": "Amazon"}, time.Now(), "field renamed")

	data, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	meta, ok := decoded["meta"].(map[string]any)
	if !ok {
		t.Fatalf("meta missing from JSON: %v", decoded)
	}
	w, ok := meta["warnings"].([]any)
	if !ok || len(w) != 1 {
		t.Fatalf("meta.warnings = %v, want 1-element array", meta["warnings"])
	}
}
