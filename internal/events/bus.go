package events

import (
	"sync"
	"time"
)

// Event represents a parsed game event from any source (log, gRPC).
type Event struct {
	Type      string            // e.g. "ship_boarded", "contract_accepted", "funds_changed"
	Source    string            // "log" or "grpc"
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
