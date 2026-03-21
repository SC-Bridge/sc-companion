package config

import (
	"os"
	"os/exec"
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

// DetectGameLog attempts to find Game.log using multiple strategies:
// 1. Windows Registry — reads RSI Launcher install path
// 2. Drive scanning — checks common install patterns on all drives
// Returns the first match found, preferring LIVE over PTU.
func DetectGameLog() string {
	if runtime.GOOS != "windows" {
		return ""
	}

	channels := []string{"LIVE", "PTU", "EPTU"}

	// Strategy 1: Windows Registry — RSI Launcher uninstall path
	if rsiRoot := detectFromRegistry(); rsiRoot != "" {
		for _, channel := range channels {
			candidate := filepath.Join(rsiRoot, "StarCitizen", channel, "Game.log")
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
		}
	}

	// Strategy 2: Scan all drive letters with common patterns
	var drives []string
	for d := 'C'; d <= 'Z'; d++ {
		root := string(d) + `:\`
		if _, err := os.Stat(root); err == nil {
			drives = append(drives, string(d))
		}
	}

	patterns := []string{
		`Program Files\Roberts Space Industries\StarCitizen`,
		`Roberts Space Industries\StarCitizen`,
		`Games\Roberts Space Industries\StarCitizen`,
		`Games\StarCitizen`,
		`Star Citizen`,
		`StarCitizen`,
		`SC\StarCitizen`,
		`RSI\StarCitizen`,
	}

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

	// Strategy 3: Walk one level deep under any found StarCitizen dirs
	for _, drive := range drives {
		for _, pattern := range patterns {
			scDir := filepath.Join(drive+`:\`, pattern)
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

// detectFromRegistry reads the RSI Launcher install location from the
// Windows registry (HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall).
func detectFromRegistry() string {
	// Use reg.exe to query — avoids CGO dependency on golang.org/x/sys/windows/registry
	out, err := exec.Command("reg", "query",
		`HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`,
		"/s", "/f", "RSI Launcher", "/d",
	).Output()
	if err != nil {
		return ""
	}

	// Parse output for UninstallString which contains the install path
	// Format: UninstallString    REG_SZ    "D:\Roberts Space Industries\RSI Launcher\Uninstall RSI Launcher.exe" /allusers
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, "UninstallString") {
			continue
		}
		// Extract the path from between quotes
		start := strings.Index(line, `"`)
		if start < 0 {
			continue
		}
		end := strings.Index(line[start+1:], `"`)
		if end < 0 {
			continue
		}
		uninstallPath := line[start+1 : start+1+end]
		// Go up two directories: "...\RSI Launcher\Uninstall RSI Launcher.exe" → "..."
		rsiRoot := filepath.Dir(filepath.Dir(uninstallPath))
		if _, err := os.Stat(rsiRoot); err == nil {
			return rsiRoot
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
