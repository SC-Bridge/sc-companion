package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds application configuration.
type Config struct {
	LogPath     string `yaml:"log_path"`
	APIEndpoint string `yaml:"api_endpoint"`
	APIToken    string `yaml:"api_token"`
	Environment string `yaml:"environment"`
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

// DetectGameLog attempts to find Game.log by scanning all available drives
// and common install patterns. Returns the first match found, preferring
// LIVE over PTU.
func DetectGameLog() string {
	if runtime.GOOS != "windows" {
		return ""
	}

	// Scan all drive letters A-Z
	var drives []string
	for d := 'C'; d <= 'Z'; d++ {
		root := string(d) + `:\`
		if _, err := os.Stat(root); err == nil {
			drives = append(drives, string(d))
		}
	}

	// Common install path patterns (relative to drive root)
	// People install SC to wildly different locations
	patterns := []string{
		// Default RSI launcher locations
		`Program Files\Roberts Space Industries\StarCitizen`,
		`Roberts Space Industries\StarCitizen`,
		// Common custom locations
		`Games\Roberts Space Industries\StarCitizen`,
		`Games\StarCitizen`,
		`Star Citizen`,
		`StarCitizen`,
		`SC\StarCitizen`,
		`RSI\StarCitizen`,
	}

	// Check LIVE first, then PTU, then EPTU
	channels := []string{"LIVE", "PTU", "EPTU"}

	for _, drive := range drives {
		for _, pattern := range patterns {
			for _, channel := range channels {
				full := filepath.Join(drive+`:\`, pattern, channel, "Game.log")
				if _, err := os.Stat(full); err == nil {
					return full
				}
			}
		}
	}

	// Last resort: recursive search for Game.log in StarCitizen directories
	for _, drive := range drives {
		for _, pattern := range patterns {
			scDir := filepath.Join(drive+`:\`, pattern)
			if _, err := os.Stat(scDir); err != nil {
				continue
			}
			// Walk one level deep looking for Game.log
			entries, err := os.ReadDir(scDir)
			if err != nil {
				continue
			}
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				candidate := filepath.Join(scDir, entry.Name(), "Game.log")
				if _, err := os.Stat(candidate); err == nil {
					return candidate
				}
			}
		}
	}

	return ""
}

// DetectedLogPath returns the auto-detected path or empty string.
// This is separate from DetectGameLog so we can show the detected path
// to the user even when they've set a manual override.
func DetectedLogPath() string {
	return DetectGameLog()
}

// ValidateLogPath checks if the given path points to a readable file.
func ValidateLogPath(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".log")
}
