package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/monarchmoney-cli/internal/auth"
)

// saveTestSession writes a valid session file for test commands that require auth.
func saveTestSession(t *testing.T, sessionPath string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(sessionPath), 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	s := auth.Session{
		Profile:   "default",
		Email:     "test@example.com",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Token:     "test-token",
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent() error = %v", err)
	}
	if err := os.WriteFile(sessionPath, data, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

func TestWriteVersion(t *testing.T) {
	t.Run("plain text", func(t *testing.T) {
		var buf bytes.Buffer
		if err := writeVersion(&buf, "default", false, false, time.Second); err != nil {
			t.Fatalf("writeVersion() error = %v", err)
		}
		got := buf.String()
		if !strings.Contains(got, "monarch version ") {
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
		var decoded map[string]any
		if err := json.Unmarshal([]byte(got), &decoded); err != nil {
			t.Fatalf("json.Unmarshal() error = %v", err)
		}
	})
}

func TestAnnotateRequiredFlags(t *testing.T) {
	newCmd := func() *cobra.Command {
		cmd := &cobra.Command{Use: "probe"}
		cmd.Flags().String("need", "", "needed value")
		cmd.Flags().String("want", "", "optional value")
		if err := cmd.MarkFlagRequired("need"); err != nil {
			t.Fatalf("MarkFlagRequired() error = %v", err)
		}
		sub := &cobra.Command{Use: "sub"}
		sub.Flags().String("subneed", "", "nested needed value")
		if err := sub.MarkFlagRequired("subneed"); err != nil {
			t.Fatalf("MarkFlagRequired() error = %v", err)
		}
		cmd.AddCommand(sub)
		return cmd
	}

	t.Run("prefixes required and recurses", func(t *testing.T) {
		root := newCmd()
		annotateRequiredFlags(root)
		if got := root.Flags().Lookup("need").Usage; got != "required: needed value" {
			t.Fatalf("need usage = %q", got)
		}
		if got := root.Flags().Lookup("want").Usage; got != "optional value" {
			t.Fatalf("want usage = %q", got)
		}
		sub, _, _ := root.Find([]string{"sub"})
		if got := sub.Flags().Lookup("subneed").Usage; got != "required: nested needed value" {
			t.Fatalf("subneed usage = %q", got)
		}
	})

	t.Run("idempotent", func(t *testing.T) {
		root := newCmd()
		annotateRequiredFlags(root)
		annotateRequiredFlags(root)
		if got := root.Flags().Lookup("need").Usage; got != "required: needed value" {
			t.Fatalf("need usage after rerun = %q", got)
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
