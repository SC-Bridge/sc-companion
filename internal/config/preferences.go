package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// SyncPreferences controls which event types are synced to the API.
type SyncPreferences struct {
	SyncEnabled map[string]bool `json:"sync_enabled"`
}

// DefaultSyncPreferences returns preferences with sync-worthy types enabled.
func DefaultSyncPreferences() *SyncPreferences {
	return &SyncPreferences{
		SyncEnabled: map[string]bool{
			// Session
			"player_login":  true,
			"server_joined": true,
			// Ship activity
			"ship_boarded":              true,
			"ship_exited":               true,
			"insurance_claim":           true,
			"insurance_claim_complete":  true,
			// Mission lifecycle
			"contract_accepted":  true,
			"contract_completed": true,
			"contract_failed":    true,
			"mission_ended":      true,
			// Location
			"location_change":      true,
			"jurisdiction_entered": true,
			// Economy
			"money_sent":           true,
			"fined":                true,
			"transaction_complete": true,
			"rewards_earned":       true,
			"refinery_complete":    true,
			// Combat/Health
			"fatal_collision": true,
		},
	}
}

// LoadSyncPreferences reads preferences from the data directory.
func LoadSyncPreferences() *SyncPreferences {
	path := filepath.Join(DataDir(), "sync-preferences.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return DefaultSyncPreferences()
	}
	prefs := &SyncPreferences{}
	if err := json.Unmarshal(data, prefs); err != nil {
		return DefaultSyncPreferences()
	}
	if prefs.SyncEnabled == nil {
		return DefaultSyncPreferences()
	}
	return prefs
}

// Save writes preferences to the data directory.
func (p *SyncPreferences) Save() error {
	path := filepath.Join(DataDir(), "sync-preferences.json")
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// IsEnabled returns whether an event type should be synced.
func (p *SyncPreferences) IsEnabled(eventType string) bool {
	if p.SyncEnabled == nil {
		return false
	}
	enabled, exists := p.SyncEnabled[eventType]
	if !exists {
		return false
	}
	return enabled
}
