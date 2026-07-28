package config

import (
	"fmt"
	"os"
	"strings"
)

func ResolveSecret(value string) (string, error) {
	const prefix = "env:"
	if !strings.HasPrefix(value, prefix) {
		return value, nil
	}
	name := strings.TrimPrefix(value, prefix)
	resolved := os.Getenv(name)
	if resolved == "" {
		return "", fmt.Errorf("value references env var %q which is not set", name)
	}
	return resolved, nil
}
