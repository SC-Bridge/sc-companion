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
