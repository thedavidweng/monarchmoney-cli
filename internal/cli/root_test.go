package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestWriteVersion(t *testing.T) {
	t.Run("plain text", func(t *testing.T) {
		var buf bytes.Buffer
		if err := writeVersion(&buf, "default", false, false, time.Second); err != nil {
			t.Fatalf("writeVersion() error = %v", err)
		}
		got := buf.String()
		if !strings.HasPrefix(got, "monarch version ") {
			t.Fatalf("writeVersion() = %q", got)
		}
	})

	t.Run("compact json", func(t *testing.T) {
		var buf bytes.Buffer
		if err := writeVersion(&buf, "default", true, false, time.Second); err != nil {
			t.Fatalf("writeVersion() error = %v", err)
		}
		var got struct {
			OK   bool `json:"ok"`
			Data struct {
				Version string `json:"version"`
				Commit  string `json:"commit"`
				Date    string `json:"date"`
			} `json:"data"`
			Meta struct {
				Command  string   `json:"command"`
				Profile  string   `json:"profile"`
				Warnings []string `json:"warnings"`
			} `json:"meta"`
		}
		if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &got); err != nil {
			t.Fatalf("json.Unmarshal() error = %v", err)
		}
		if !got.OK || got.Data.Version == "" || got.Meta.Command != "version" || got.Meta.Profile != "default" {
			t.Fatalf("writeVersion() = %#v", got)
		}
	})

	t.Run("pretty json", func(t *testing.T) {
		var buf bytes.Buffer
		if err := writeVersion(&buf, "default", true, true, time.Second); err != nil {
			t.Fatalf("writeVersion() error = %v", err)
		}
		got := buf.String()
		if !strings.Contains(got, "\n  \"ok\"") {
			t.Fatalf("writeVersion() = %q", got)
		}
		var decoded map[string]interface{}
		if err := json.Unmarshal([]byte(got), &decoded); err != nil {
			t.Fatalf("json.Unmarshal() error = %v", err)
		}
	})
}

func TestEnvelopeWithWarnings(t *testing.T) {
	originalProfile := profile
	profile = "default"
	defer func() { profile = originalProfile }()

	env := envelopeWithWarnings("transactions.list", map[string]string{"status": "ok"}, time.Now(), "uses legacy Monarch GraphQL root field: allTransactions")
	if len(env.Meta.Warnings) != 1 || env.Meta.Warnings[0] == "" {
		t.Fatalf("envelopeWithWarnings() = %#v", env.Meta.Warnings)
	}
	data, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if !strings.Contains(string(data), `"warnings":["uses legacy Monarch GraphQL root field: allTransactions"]`) {
		t.Fatalf("envelopeWithWarnings() = %s", string(data))
	}
}
