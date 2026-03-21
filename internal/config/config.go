package config

import (
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
)

// Config holds application configuration.
type Config struct {
	LogPath     string `yaml:"log_path"`
	APIEndpoint string `yaml:"api_endpoint"`
	APIToken    string `yaml:"api_token"`
}

// Default returns a config with sensible defaults.
func Default() *Config {
	return &Config{
		APIEndpoint: "https://scbridge.app/api",
	}
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
