package config

import (
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
)

// Config holds application configuration.
type Config struct {
	LogPath      string `yaml:"log_path"`
	APIEndpoint  string `yaml:"api_endpoint"`
	APIToken     string `yaml:"api_token"`
	Environment  string `yaml:"environment"`
	ProxyEnabled bool   `yaml:"proxy_enabled"`
	ProxyPort    int    `yaml:"proxy_port"`
}

// Default returns a config with sensible defaults.
func Default() *Config {
	return &Config{
		APIEndpoint: "https://scbridge.app/api",
		Environment: "production",
	}
}

// EndpointForEnv returns the API endpoint for the given environment.
func EndpointForEnv(env string) string {
	if env == "staging" {
		return "https://staging.scbridge.app/api"
	}
	return "https://scbridge.app/api"
}

// Save writes the config to a YAML file.
func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// DataDir returns the application data directory.
func DataDir() string {
	if runtime.GOOS == "windows" {
		if appData := os.Getenv("APPDATA"); appData != "" {
			return filepath.Join(appData, "SCBridge")
		}
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".scbridge")
}

// Load reads config from a YAML file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := Default()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	// Normalize empty environment to default
	if cfg.Environment == "" {
		cfg.Environment = "production"
	}
	return cfg, nil
}

// DetectGameLog attempts to find Game.log in common SC install locations.
func DetectGameLog() string {
	if runtime.GOOS != "windows" {
		return ""
	}

	// Common SC install drives
	drives := []string{"C", "D", "E", "F"}
	paths := []string{
		`Program Files\Roberts Space Industries\StarCitizen\LIVE\Game.log`,
		`Roberts Space Industries\StarCitizen\LIVE\Game.log`,
		`Games\Roberts Space Industries\StarCitizen\LIVE\Game.log`,
		`Program Files\Roberts Space Industries\StarCitizen\PTU\Game.log`,
		`Roberts Space Industries\StarCitizen\PTU\Game.log`,
	}

	for _, drive := range drives {
		for _, p := range paths {
			full := filepath.Join(drive+`:\`, p)
			if _, err := os.Stat(full); err == nil {
				return full
			}
		}
	}
	return ""
}
