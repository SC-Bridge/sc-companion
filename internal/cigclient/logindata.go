package cigclient

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// LoginData matches the loginData.json written by Star Citizen at launch.
type LoginData struct {
	Username    string      `json:"username"`
	Token       string      `json:"token"`
	AuthToken   string      `json:"auth_token"`
	StarNetwork StarNetwork `json:"star_network"`
}

// StarNetwork holds the CIG backend connection info.
type StarNetwork struct {
	ServicesEndpoint string `json:"services_endpoint"`
	Hostname         string `json:"hostname"`
	Port             int    `json:"port"`
}

// FindLoginData searches common SC install locations for loginData.json.
func FindLoginData() (string, error) {
	if runtime.GOOS != "windows" {
		return "", fmt.Errorf("loginData.json detection only supported on Windows")
	}

	drives := []string{"C", "D", "E", "F"}
	variants := []string{"LIVE", "PTU", "EPTU"}

	for _, drive := range drives {
		for _, variant := range variants {
			paths := []string{
				filepath.Join(drive+`:\`, "Roberts Space Industries", "StarCitizen", variant, "loginData.json"),
				filepath.Join(drive+`:\`, "Program Files", "Roberts Space Industries", "StarCitizen", variant, "loginData.json"),
				filepath.Join(drive+`:\`, "Games", "Roberts Space Industries", "StarCitizen", variant, "loginData.json"),
			}
			for _, p := range paths {
				if _, err := os.Stat(p); err == nil {
					return p, nil
				}
			}
		}
	}
	return "", fmt.Errorf("loginData.json not found")
}

// ReadLoginData reads and parses loginData.json from the given path.
func ReadLoginData(path string) (*LoginData, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read loginData.json: %w", err)
	}
	var ld LoginData
	if err := json.Unmarshal(data, &ld); err != nil {
		return nil, fmt.Errorf("parse loginData.json: %w", err)
	}
	if ld.StarNetwork.ServicesEndpoint == "" {
		return nil, fmt.Errorf("loginData.json has no services_endpoint")
	}
	return &ld, nil
}

// WatchLoginData watches for loginData.json to appear or change.
// Calls onChange when it's found or updated. Blocks until ctx is done.
func WatchLoginData(path string, onChange func(*LoginData)) {
	var lastMod time.Time
	for {
		info, err := os.Stat(path)
		if err == nil && info.ModTime() != lastMod {
			lastMod = info.ModTime()
			ld, err := ReadLoginData(path)
			if err != nil {
				slog.Warn("failed to read loginData.json", "error", err)
			} else {
				onChange(ld)
			}
		}
		time.Sleep(2 * time.Second)
	}
}
