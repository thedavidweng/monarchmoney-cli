package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/viper"
)

func TestDefaultAuditDir(t *testing.T) {
	home, _ := os.UserHomeDir()
	if got, want := DefaultAuditDir(), filepath.Join(home, ".monarchmoney-cli", "audit"); got != want {
		t.Fatalf("DefaultAuditDir() = %q, want %q", got, want)
	}
}

func TestLoadIncludesAllDefaults(t *testing.T) {
	viper.Reset()
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Profile != "default" || cfg.APIEndpoint != "https://api.monarch.com/graphql" || cfg.Timeout != 30*time.Second || cfg.ReadOnly || cfg.SessionPath == "" || !cfg.AuditLog || cfg.CachePath == "" {
		t.Fatalf("Load() returned unexpected config: %#v", cfg)
	}
}

func TestLoadReturnsUnmarshalError(t *testing.T) {
	original := unmarshalConfig
	unmarshalConfig = func(any, ...viper.DecoderConfigOption) error { return errors.New("decode failed") }
	defer func() { unmarshalConfig = original }()

	viper.Reset()
	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want failure")
	}
}
