package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPathExpansion(t *testing.T) {
	home, _ := os.UserHomeDir()

	expected := filepath.Join(home, ".monarchmoney-cli", "config.yaml")
	assert.Equal(t, expected, DefaultConfigPath())
}
