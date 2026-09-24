package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadReturnsFileErrorButUsableConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("- not\n- a\n- mapping\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := Load(path)
	if err == nil {
		t.Fatal("Load() error = nil, want parse failure")
	}
	if cfg == nil {
		t.Fatal("Load() cfg = nil, want defaults even on parse error")
	}
	if cfg.APIEndpoint != "https://api.monarch.com/graphql" {
		t.Fatalf("cfg.APIEndpoint = %q, want default (env/defaults still applied)", cfg.APIEndpoint)
	}
}
