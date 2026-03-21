package events

import (
	"sync"
	"time"
)

// Event represents a parsed game event from the log tailer.
type Event struct {
	Type      string            // e.g. "ship_boarded", "contract_accepted", "funds_changed"
	Source    string            // "log"
	Timestamp time.Time
	Data      map[string]string // key-value pairs specific to the event type
}

// SyncWorthyTypes are events worth syncing to the SC Bridge API.
// Everything else is local-only (companion event feed / debug).
var SyncWorthyTypes = map[string]bool{
	// Session
	"player_login":  true,
	"server_joined": true,
	// Ship activity
	"ship_boarded":   true,
	"ship_exited":    true,
	"insurance_claim": true,
	"insurance_claim_complete": true,
	// Mission lifecycle
	"contract_accepted":  true,
	"contract_completed": true,
	"contract_failed":    true,
	"mission_ended":      true,
	// Location (heartbeat enrichment)
	"location_change":       true,
	"jurisdiction_entered":  true,
	// Economy
	"money_sent":           true,
	"fined":                true,
	"transaction_complete": true,
}

// IsSyncWorthy returns true if this event type should be synced to the API.
func (e Event) IsSyncWorthy() bool {
	return SyncWorthyTypes[e.Type]
}

// EventCategoryEntry describes a single event type within a category.
type EventCategoryEntry struct {
	Type  string `json:"type"`
	Label string `json:"label"`
}

// EventCategory groups related event types.
type EventCategory struct {
	Name   string               `json:"name"`
	Events []EventCategoryEntry `json:"events"`
}

// EventCategories returns all 40 event types grouped into 8 categories.
func EventCategories() []EventCategory {
	return []EventCategory{
		{
			Name: "Session",
			Events: []EventCategoryEntry{
				{Type: "player_login", Label: "Player Login"},
				{Type: "server_joined", Label: "Server Joined"},
			},
		},
		{
			Name: "Ships",
			Events: []EventCategoryEntry{
				{Type: "ship_boarded", Label: "Ship Boarded"},
				{Type: "ship_exited", Label: "Ship Exited"},
				{Type: "insurance_claim", Label: "Insurance Claim"},
				{Type: "insurance_claim_complete", Label: "Claim Complete"},
				{Type: "vehicle_impounded", Label: "Vehicle Impounded"},
				{Type: "hangar_ready", Label: "Hangar Ready"},
				{Type: "ship_list_fetched", Label: "Ship List Fetched"},
				{Type: "ships_loaded", Label: "Ships Loaded"},
				{Type: "entitlement_reconciliation", Label: "Entitlement Reconciliation"},
			},
		},
		{
			Name: "Missions",
			Events: []EventCategoryEntry{
				{Type: "contract_accepted", Label: "Contract Accepted"},
				{Type: "contract_completed", Label: "Contract Completed"},
				{Type: "contract_failed", Label: "Contract Failed"},
				{Type: "contract_available", Label: "Contract Available"},
				{Type: "mission_ended", Label: "Mission Ended"},
				{Type: "end_mission", Label: "End Mission"},
				{Type: "new_objective", Label: "New Objective"},
			},
		},
		{
			Name: "Location",
			Events: []EventCategoryEntry{
				{Type: "location_change", Label: "Location Change"},
				{Type: "jurisdiction_entered", Label: "Jurisdiction Entered"},
				{Type: "armistice_entered", Label: "Armistice Entered"},
				{Type: "armistice_exited", Label: "Armistice Exited"},
				{Type: "monitored_space_entered", Label: "Monitored Space Entered"},
				{Type: "monitored_space_exited", Label: "Monitored Space Exited"},
			},
		},
		{
			Name: "Quantum Travel",
			Events: []EventCategoryEntry{
				{Type: "qt_target_selected", Label: "Target Selected"},
				{Type: "qt_destination_selected", Label: "Destination Selected"},
				{Type: "qt_fuel_requested", Label: "Fuel Requested"},
				{Type: "qt_arrived", Label: "Arrived"},
			},
		},
		{
			Name: "Economy",
			Events: []EventCategoryEntry{
				{Type: "money_sent", Label: "Money Sent"},
				{Type: "money_sent_pending", Label: "Money Pending"},
				{Type: "fined", Label: "Fined"},
				{Type: "transaction_complete", Label: "Transaction Complete"},
				{Type: "rewards_earned", Label: "Rewards Earned"},
				{Type: "refinery_complete", Label: "Refinery Complete"},
				{Type: "blueprint_received", Label: "Blueprint Received"},
			},
		},
		{
			Name: "Combat & Health",
			Events: []EventCategoryEntry{
				{Type: "injury", Label: "Injury"},
				{Type: "incapacitated", Label: "Incapacitated"},
				{Type: "fatal_collision", Label: "Fatal Collision"},
				{Type: "crimestat_increased", Label: "CrimeStat Increased"},
				{Type: "emergency_services", Label: "Emergency Services"},
			},
		},
		{
			Name: "System",
			Events: []EventCategoryEntry{
				{Type: "money_amount", Label: "Money Amount (internal)"},
			},
		},
	}
}

// Handler processes events from the bus.
type Handler func(Event)

// Bus is a simple pub/sub event bus. All parsed events flow through it.
type Bus struct {
	mu       sync.RWMutex
	handlers []Handler
}

// NewBus creates a new event bus.
func NewBus() *Bus {
	return &Bus{}
}

// Subscribe registers a handler that receives all events.
func (b *Bus) Subscribe(h Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers = append(b.handlers, h)
}

// Publish sends an event to all subscribers.
func (b *Bus) Publish(evt Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, h := range b.handlers {
		h(evt)
	}
}
