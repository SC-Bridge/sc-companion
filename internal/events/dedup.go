package events

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// Deduplicator suppresses duplicate events within a cooldown window.
// Game.log often emits the same notification multiple times (Added, Next, StartFade, Remove).
type Deduplicator struct {
	mu       sync.Mutex
	seen     map[string]time.Time
	cooldown time.Duration
}

// NewDeduplicator creates a deduplicator with the given cooldown window.
func NewDeduplicator(cooldown time.Duration) *Deduplicator {
	d := &Deduplicator{
		seen:     make(map[string]time.Time),
		cooldown: cooldown,
	}
	// Periodic cleanup of stale entries
	go d.cleanup()
	return d
}

// IsDuplicate returns true if this event was seen recently.
func (d *Deduplicator) IsDuplicate(evt Event) bool {
	key := eventKey(evt)

	d.mu.Lock()
	defer d.mu.Unlock()

	if lastSeen, ok := d.seen[key]; ok {
		if time.Since(lastSeen) < d.cooldown {
			return true
		}
	}
	d.seen[key] = time.Now()
	return false
}

// eventKey creates a fingerprint from an event's type + data.
func eventKey(evt Event) string {
	h := sha256.New()
	h.Write([]byte(evt.Type))

	// Sort keys for deterministic hashing
	keys := make([]string, 0, len(evt.Data))
	for k := range evt.Data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		h.Write([]byte(k))
		h.Write([]byte(evt.Data[k]))
	}

	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}

func (d *Deduplicator) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		d.mu.Lock()
		cutoff := time.Now().Add(-d.cooldown * 2)
		for k, v := range d.seen {
			if v.Before(cutoff) {
				delete(d.seen, k)
			}
		}
		d.mu.Unlock()
	}
}

// CoalesceMultiLine handles multi-line notifications like money transfers.
// It accumulates "money_sent_pending" events and merges them with the next "money_amount" event.
type CoalesceMultiLine struct {
	mu      sync.Mutex
	pending *Event
}

// NewCoalesceMultiLine creates a multi-line event coalescer.
func NewCoalesceMultiLine() *CoalesceMultiLine {
	return &CoalesceMultiLine{}
}

// Process takes an event and returns a possibly coalesced event.
// Returns (event, true) if an event should be emitted, (_, false) if it was absorbed.
func (c *CoalesceMultiLine) Process(evt Event) (Event, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	switch evt.Type {
	case "money_sent_pending":
		// Start accumulating — don't emit yet
		c.pending = &evt
		return Event{}, false

	case "money_amount":
		if c.pending != nil && c.pending.Type == "money_sent_pending" {
			// Merge: combine recipient from pending with amount from this event
			merged := Event{
				Type:      "money_sent",
				Source:    "log",
				Timestamp: c.pending.Timestamp,
				Data: map[string]string{
					"recipient": c.pending.Data["recipient"],
					"amount":    evt.Data["amount"],
					"currency":  "aUEC",
				},
			}
			c.pending = nil
			return merged, true
		}
		// No pending — emit as-is (orphaned amount line)
		return evt, true

	default:
		// Any non-money event clears pending state (timeout)
		if c.pending != nil {
			// The pending money_sent never got an amount — drop it
			c.pending = nil
		}
		return evt, true
	}
}

// FilterType returns a handler that only processes events of certain types.
func FilterType(types []string, h Handler) Handler {
	set := make(map[string]bool, len(types))
	for _, t := range types {
		set[t] = true
	}
	return func(evt Event) {
		if set[evt.Type] {
			h(evt)
		}
	}
}

// IgnoreType returns a handler that ignores events matching certain prefixes.
func IgnoreType(prefixes []string, h Handler) Handler {
	return func(evt Event) {
		for _, p := range prefixes {
			if strings.HasPrefix(evt.Type, p) {
				return
			}
		}
		h(evt)
	}
}
