// Package config loads runtime configuration from environment variables.
package config

import (
	"os"
	"strings"
)

// Config holds environment-sourced runtime settings.
type Config struct {
	// SettingsFile is the path to setting.yml (or any YAML settings file).
	SettingsFile string
}

// Load reads configuration from the environment, applying sensible defaults.
func Load() *Config {
	return &Config{
		SettingsFile: env("H2H_SETTING_FILE", "./setting.yml"),
	}
}

func env(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		return v
	}
	return fallback
}
