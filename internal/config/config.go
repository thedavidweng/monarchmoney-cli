package config

import (
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

// Config represents the application configuration.
type Config struct {
	Profile     string        `mapstructure:"profile"`
	APIEndpoint string        `mapstructure:"api_endpoint"`
	Output      string        `mapstructure:"output"`
	Timeout     time.Duration `mapstructure:"timeout"`
	ReadOnly    bool          `mapstructure:"read_only"`
	SessionPath string        `mapstructure:"session_path"`
	AuditLog    bool          `mapstructure:"audit_log"`
	CachePath   string        `mapstructure:"cache_path"`
}

// Load loads the configuration from viper.
func Load() (*Config, error) {
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	// Set defaults if not provided
	if cfg.Profile == "" {
		cfg.Profile = "default"
	}
	if cfg.APIEndpoint == "" {
		cfg.APIEndpoint = "https://api.monarch.com/graphql"
	}
	if cfg.SessionPath == "" {
		cfg.SessionPath = DefaultSessionPath()
	}
	if cfg.CachePath == "" {
		cfg.CachePath = filepath.Join(DefaultCacheDir(), "monarch.sqlite")
	}

	return &cfg, nil
}
