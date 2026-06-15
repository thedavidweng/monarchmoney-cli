package config

import (
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestConfigPrecedence(t *testing.T) {
	viper.Reset()
	os.Setenv("MONARCH_PROFILE", "env-profile") //nolint:errcheck // test env setup
	defer os.Unsetenv("MONARCH_PROFILE")        //nolint:errcheck // test env cleanup

	// Precedence: CLI flags (passed via viper.Set) > Env vars > Config file > Defaults

	// Default
	cfg, _ := Load()
	assert.Equal(t, "env-profile", cfg.Profile) // Env takes precedence over default "default"

	// Flag override
	viper.Set("profile", "flag-profile")
	cfg, _ = Load()
	assert.Equal(t, "flag-profile", cfg.Profile)
}
