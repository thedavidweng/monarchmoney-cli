package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Profile     string        `yaml:"profile"`
	APIEndpoint string        `yaml:"api_endpoint"`
	Output      string        `yaml:"output"`
	Timeout     time.Duration `yaml:"timeout"`
	ReadOnly    bool          `yaml:"read_only"`
	SessionPath string        `yaml:"session_path"`
	AuditLog    bool          `yaml:"audit_log"`
	CachePath   string        `yaml:"cache_path"`
}

func defaults() *Config {
	return &Config{
		Profile:     "default",
		APIEndpoint: "https://api.monarch.com/graphql",
		Timeout:     30 * time.Second,
		ReadOnly:    false,
		SessionPath: DefaultSessionPath(),
		AuditLog:    true,
		CachePath:   DefaultCachePath(),
	}
}

func Load(path string) (*Config, error) {
	cfg := defaults()

	if path == "" {
		path = DefaultConfigPath()
	}
	fileErr := applyFileFromPath(cfg, path)
	applyEnv(cfg)
	return cfg, fileErr
}

func applyFileFromPath(cfg *Config, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read config %s: %w", path, err)
	}
	raw := map[string]any{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("parse config %s: %w", path, err)
	}
	applyFile(cfg, raw)
	return nil
}

func applyFile(cfg *Config, raw map[string]any) {
	if v, ok := raw["profile"].(string); ok {
		cfg.Profile = v
	}
	if v, ok := raw["api_endpoint"].(string); ok {
		cfg.APIEndpoint = v
	}
	if v, ok := raw["output"].(string); ok {
		cfg.Output = v
	}
	if v, ok := raw["timeout"]; ok {
		if d, dok := coerceDuration(v); dok {
			cfg.Timeout = d
		}
	}
	if v, ok := raw["read_only"].(bool); ok {
		cfg.ReadOnly = v
	}
	if v, ok := raw["session_path"].(string); ok {
		cfg.SessionPath = v
	}
	if v, ok := raw["audit_log"].(bool); ok {
		cfg.AuditLog = v
	}
	if v, ok := raw["cache_path"].(string); ok {
		cfg.CachePath = v
	}
}

func applyEnv(cfg *Config) {
	if v := os.Getenv("MONARCH_PROFILE"); v != "" {
		cfg.Profile = v
	}
	if v := os.Getenv("MONARCH_API_ENDPOINT"); v != "" {
		cfg.APIEndpoint = v
	}
	if v := os.Getenv("MONARCH_OUTPUT"); v != "" {
		cfg.Output = v
	}
	if v := os.Getenv("MONARCH_TIMEOUT"); v != "" {
		if d, ok := coerceDuration(v); ok {
			cfg.Timeout = d
		}
	}
	if v := os.Getenv("MONARCH_READ_ONLY"); v != "" {
		cfg.ReadOnly = ParseBool(v)
	}
	if v := os.Getenv("MONARCH_SESSION_PATH"); v != "" {
		cfg.SessionPath = v
	}
	if v := os.Getenv("MONARCH_AUDIT_LOG"); v != "" {
		cfg.AuditLog = ParseBool(v)
	}
	if v := os.Getenv("MONARCH_CACHE_PATH"); v != "" {
		cfg.CachePath = v
	}
}

func coerceDuration(v any) (time.Duration, bool) {
	switch t := v.(type) {
	case string:
		if d, err := time.ParseDuration(t); err == nil {
			return d, true
		}
		if n, err := strconv.ParseInt(t, 10, 64); err == nil {
			return time.Duration(n), true
		}
	case int:
		return time.Duration(t), true
	case int64:
		return time.Duration(t), true
	case float64:
		return time.Duration(int64(t)), true
	}
	return 0, false
}

func ParseBool(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes":
		return true
	default:
		return false
	}
}
