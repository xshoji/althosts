// Package config loads and writes ~/.althosts/config.yaml.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
)

// Config represents the contents of config.yaml.
type Config struct {
	HostsPath string `yaml:"hosts_path"`
	FlushDNS  bool   `yaml:"flush_dns"`
	Editor    string `yaml:"editor"`
}

// Default returns the default config for the current platform.
func Default() Config {
	return Config{
		HostsPath: DefaultHostsPath(),
		FlushDNS:  true,
		Editor:    "",
	}
}

// Load reads config.yaml from path. If the file does not exist, defaults are returned.
func Load(path string) (Config, error) {
	cfg := Default()
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("read config: %w", err)
	}
	if len(b) == 0 {
		return cfg, nil
	}
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}
	cfg.fillDefaults()
	return cfg, nil
}

// Save writes the config to path with 0o644.
func Save(path string, cfg Config) error {
	b, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	return os.WriteFile(path, b, 0o644)
}

func (c *Config) fillDefaults() {
	if c.HostsPath == "" {
		c.HostsPath = DefaultHostsPath()
	}
}

// DefaultHostsPath returns the platform's default hosts file path.
func DefaultHostsPath() string {
	if runtime.GOOS == "windows" {
		return `C:\Windows\System32\drivers\etc\hosts`
	}
	return "/etc/hosts"
}

// IsDefaultHostsPath reports whether path points to the platform default hosts file.
func IsDefaultHostsPath(path string) bool {
	return canonicalPath(path) == canonicalPath(DefaultHostsPath())
}

func canonicalPath(path string) string {
	if real, err := filepath.EvalSymlinks(path); err == nil {
		path = real
	}
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	return filepath.Clean(path)
}
