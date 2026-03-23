package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds application configuration.
type Config struct {
	LogPath        string `yaml:"log_path"`
	APIEndpoint    string `yaml:"api_endpoint"`
	APIToken       string `yaml:"api_token"`
	Environment    string `yaml:"environment"`
	MinimizeToTray bool   `yaml:"minimize_to_tray"`
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
// 0. RSI Launcher log — reads exact channel paths from the launcher's own log file
// 1. Running process — queries the live StarCitizen.exe process path via WMIC
// 2. Windows Registry — reads RSI Launcher install path (HKLM and HKCU)
// 3. Launcher exe — searches drives for StarCitizen_Launcher.exe
// 4. Drive scanning — checks common install patterns on all drives
// Returns the first match found, preferring the most recently launched channel.
func DetectGameLog() string {
	if runtime.GOOS != "windows" {
		return ""
	}

	channels := []string{"LIVE", "PTU", "EPTU"}

	// Strategy 0: RSI Launcher log — most reliable, works for any channel name
	if path := detectFromLauncherLog(); path != "" {
		return path
	}

	// Strategy 1: Query running StarCitizen.exe process (most reliable when game is open)
	if path := detectFromRunningProcess(); path != "" {
		return path
	}

	// Strategy 1: Windows Registry — RSI Launcher uninstall path (HKLM and HKCU)
	for _, hive := range []string{`HKLM`, `HKCU`} {
		if rsiRoot := detectFromRegistryHive(hive); rsiRoot != "" {
			scDir := filepath.Join(rsiRoot, "StarCitizen")
			for _, channel := range channels {
				candidate := filepath.Join(scDir, channel, "Game.log")
				if _, err := os.Stat(candidate); err == nil {
					return candidate
				}
			}
			// Walk one level in case channel name is non-standard
			if entries, err := os.ReadDir(scDir); err == nil {
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
	}

	// Strategy 2: Search drives for StarCitizen_Launcher.exe to locate RSI root
	if path := detectFromLauncherExe(channels); path != "" {
		return path
	}

	// Strategy 3: Scan all drive letters with common patterns
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

	// Strategy 4: Walk one level deep under any found StarCitizen dirs
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

// detectFromLauncherLog reads the RSI Launcher's own log file to find the
// most recently launched channel path. The launcher writes lines like:
//
//	[Launcher::launch] Launching Star Citizen LIVE from (C:\Roberts Space Industries\StarCitizen\LIVE)
//
// This is the most reliable strategy: it works for any channel name (LIVE,
// PTU, EPTU, HOTFIX, …) regardless of install location.
func detectFromLauncherLog() string {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		return ""
	}

	// Compiled once — matches the exact path in parentheses
	re := regexp.MustCompile(`\[Launcher::launch\] Launching Star Citizen \S+ from \(([^)]+)\)`)

	// log.log is current; log.old.log is the previous rotation
	logFiles := []string{
		filepath.Join(appData, "rsilauncher", "logs", "log.log"),
		filepath.Join(appData, "rsilauncher", "logs", "log.old.log"),
	}

	for _, logFile := range logFiles {
		data, err := os.ReadFile(logFile)
		if err != nil {
			continue
		}

		// Collect all matches and take the last one — most recently launched
		matches := re.FindAllSubmatch(data, -1)
		if len(matches) == 0 {
			continue
		}

		channelDir := string(matches[len(matches)-1][1])
		candidate := filepath.Join(channelDir, "Game.log")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	return ""
}

// detectFromRunningProcess queries WMIC for a running StarCitizen.exe and
// derives the Game.log path from its executable location.
// StarCitizen.exe lives at <channel_dir>\Bin64\StarCitizen.exe so Game.log
// is two directories up at <channel_dir>\Game.log.
func detectFromRunningProcess() string {
	out, err := exec.Command("wmic", "process",
		"where", "Name='StarCitizen.exe'",
		"get", "ExecutablePath",
	).Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.EqualFold(line, "ExecutablePath") {
			continue
		}
		// <channel_dir>\Bin64\StarCitizen.exe → up two dirs → <channel_dir>
		channelDir := filepath.Dir(filepath.Dir(line))
		candidate := filepath.Join(channelDir, "Game.log")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

// detectFromLauncherExe searches drives for StarCitizen_Launcher.exe to
// locate the RSI root directory and then finds Game.log from there.
func detectFromLauncherExe(channels []string) string {
	launcherRelPaths := []string{
		`Roberts Space Industries\RSI Launcher\StarCitizen_Launcher.exe`,
		`Program Files\Roberts Space Industries\RSI Launcher\StarCitizen_Launcher.exe`,
		`RSI Launcher\StarCitizen_Launcher.exe`,
	}
	for d := 'C'; d <= 'Z'; d++ {
		root := string(d) + `:\`
		if _, err := os.Stat(root); err != nil {
			continue
		}
		for _, rel := range launcherRelPaths {
			exePath := filepath.Join(root, rel)
			if _, err := os.Stat(exePath); err != nil {
				continue
			}
			// RSI root is two dirs up from the exe: ...\RSI Launcher\StarCitizen_Launcher.exe
			rsiRoot := filepath.Dir(filepath.Dir(exePath))
			scDir := filepath.Join(rsiRoot, "StarCitizen")
			for _, channel := range channels {
				candidate := filepath.Join(scDir, channel, "Game.log")
				if _, err := os.Stat(candidate); err == nil {
					return candidate
				}
			}
			// Walk in case channel name is non-standard
			if entries, err := os.ReadDir(scDir); err == nil {
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
	}
	return ""
}

// detectFromRegistryHive reads the RSI Launcher install location from the
// Windows registry for the given hive (HKLM or HKCU).
func detectFromRegistryHive(hive string) string {
	key := hive + `\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`
	out, err := exec.Command("reg", "query", key, "/s").Output()
	if err != nil {
		return ""
	}

	// Parse output for UninstallString values that reference RSI Launcher.
	// Format: UninstallString    REG_SZ    "D:\Roberts Space Industries\RSI Launcher\Uninstall RSI Launcher.exe" /allusers
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, "UninstallString") {
			continue
		}
		if !strings.Contains(strings.ToLower(line), "rsi launcher") {
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
