package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/SC-Bridge/sc-companion/internal/config"
)

// AuthInfo stores the companion app's session with SC Bridge.
type AuthInfo struct {
	SessionToken string `json:"session_token"`
	Handle       string `json:"handle"`
	ConnectedAt  string `json:"connected_at"`
	Endpoint     string `json:"endpoint"`
}

// authFilePath returns the path to auth.json.
func authFilePath() string {
	return filepath.Join(config.DataDir(), "auth.json")
}

// Load reads auth info from disk. Returns nil if not found or invalid.
func Load() *AuthInfo {
	data, err := os.ReadFile(authFilePath())
	if err != nil {
		return nil
	}
	info := &AuthInfo{}
	if err := json.Unmarshal(data, info); err != nil {
		return nil
	}
	if info.SessionToken == "" {
		return nil
	}
	return info
}

// Save writes auth info to disk with restrictive permissions.
func Save(info *AuthInfo) error {
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(authFilePath(), data, 0600)
}

// Clear removes the auth file.
func Clear() error {
	err := os.Remove(authFilePath())
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// NewAuthInfo creates an AuthInfo for a successful connection.
func NewAuthInfo(sessionToken, handle, endpoint string) *AuthInfo {
	return &AuthInfo{
		SessionToken: sessionToken,
		Handle:       handle,
		ConnectedAt:  time.Now().UTC().Format(time.RFC3339),
		Endpoint:     endpoint,
	}
}
