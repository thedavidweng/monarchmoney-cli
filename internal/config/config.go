package config

import (
	"time"

	"github.com/spf13/viper"
)

var unmarshalConfig = func(rawVal any, opts ...viper.DecoderConfigOption) error {
	return viper.Unmarshal(rawVal, opts...)
}

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
	viper.SetEnvPrefix("MONARCH")
	viper.AutomaticEnv()

	// Set defaults in viper
	viper.SetDefault("profile", "default")
	viper.SetDefault("api_endpoint", "https://api.monarch.com/graphql")
	viper.SetDefault("timeout", 30*time.Second)
	viper.SetDefault("read_only", false)
	viper.SetDefault("session_path", DefaultSessionPath())
	viper.SetDefault("audit_log", true)
	viper.SetDefault("cache_path", DefaultCachePath())

	var cfg Config
	if err := unmarshalConfig(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
