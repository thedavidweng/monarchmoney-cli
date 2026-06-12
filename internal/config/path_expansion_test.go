package config

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPathExpansion(t *testing.T) {
	expected := filepath.Join(userConfigDir(), configSubDir, "config.yaml")
	assert.Equal(t, expected, DefaultConfigPath())
}
