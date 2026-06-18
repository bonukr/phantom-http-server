// Package settings loads server configuration from a YAML settings file.
package settings

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Settings is the root YAML configuration (setting.yml).
type Settings struct {
	Server ServerConfig `yaml:"server"`
	Log    LogConfig    `yaml:"log"`
	APIs   []APIConfig  `yaml:"apis"`
}

// ServerConfig holds HTTP/HTTPS listener settings.
type ServerConfig struct {
	Port int        `yaml:"port"`
	TLS  TLSConfig  `yaml:"tls"`
}

// TLSConfig enables HTTPS when enabled is true.
type TLSConfig struct {
	Enabled  bool   `yaml:"enabled"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

// LogConfig holds application log settings.
type LogConfig struct {
	File  string `yaml:"file"`
	Level string `yaml:"level"`
}

// APIConfig defines a virtual REST endpoint to expose.
type APIConfig struct {
	Path        string   `yaml:"path"`
	Methods     []string `yaml:"methods"`
	Description string   `yaml:"description"`
}

// Load reads and validates settings from the given YAML file path.
func Load(path string) (*Settings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read settings: %w", err)
	}

	var s Settings
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse settings: %w", err)
	}

	if err := s.applyDefaults(path); err != nil {
		return nil, err
	}
	if err := s.validate(); err != nil {
		return nil, err
	}
	return &s, nil
}

func (s *Settings) applyDefaults(settingsPath string) error {
	if s.Server.Port == 0 {
		s.Server.Port = 8080
	}
	if strings.TrimSpace(s.Log.File) == "" {
		s.Log.File = "./logs/phantom-http-server.log"
	}
	if strings.TrimSpace(s.Log.Level) == "" {
		s.Log.Level = "info"
	}

	// Resolve relative TLS cert paths against the settings file directory.
	base := filepath.Dir(settingsPath)
	if s.Server.TLS.Enabled {
		if !filepath.IsAbs(s.Server.TLS.CertFile) {
			s.Server.TLS.CertFile = filepath.Join(base, s.Server.TLS.CertFile)
		}
		if !filepath.IsAbs(s.Server.TLS.KeyFile) {
			s.Server.TLS.KeyFile = filepath.Join(base, s.Server.TLS.KeyFile)
		}
	}

	if len(s.APIs) == 0 {
		s.APIs = []APIConfig{
			{
				Path:        "/alert-manager/hook1",
				Methods:     []string{"GET", "POST", "PUT", "DELETE"},
				Description: "Alert Manager webhook hook 1",
			},
		}
	}

	for i := range s.APIs {
		s.APIs[i].Path = normalizePath(s.APIs[i].Path)
		if len(s.APIs[i].Methods) == 0 {
			s.APIs[i].Methods = []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
		}
		for j := range s.APIs[i].Methods {
			s.APIs[i].Methods[j] = strings.ToUpper(strings.TrimSpace(s.APIs[i].Methods[j]))
		}
	}
	return nil
}

func (s *Settings) validate() error {
	if s.Server.Port < 1 || s.Server.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535")
	}
	if s.Server.TLS.Enabled {
		if strings.TrimSpace(s.Server.TLS.CertFile) == "" || strings.TrimSpace(s.Server.TLS.KeyFile) == "" {
			return fmt.Errorf("server.tls cert_file and key_file are required when tls.enabled is true")
		}
	}
	seen := make(map[string]struct{})
	for _, api := range s.APIs {
		if api.Path == "" || api.Path == "/" {
			return fmt.Errorf("api path must not be empty or root")
		}
		if _, ok := seen[api.Path]; ok {
			return fmt.Errorf("duplicate api path: %s", api.Path)
		}
		seen[api.Path] = struct{}{}
	}
	return nil
}

func normalizePath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return p
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return strings.TrimSuffix(p, "/")
}
