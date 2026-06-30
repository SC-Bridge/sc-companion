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
	mu            sync.Mutex
	seen          map[string]time.Time
	cooldown      time.Duration
	maxCooldown   time.Duration
	typeCooldowns map[string]time.Duration
}

// NewDeduplicator creates a deduplicator with the given default cooldown window.
func NewDeduplicator(cooldown time.Duration) *Deduplicator {
	d := &Deduplicator{
		seen:          make(map[string]time.Time),
		cooldown:      cooldown,
		maxCooldown:   cooldown,
		typeCooldowns: make(map[string]time.Duration),
	}
	// Periodic cleanup of stale entries
	go d.cleanup()
	return d
}

// SetTypeCooldown overrides the dedup window for a specific event type.
// Use this for notifications that repeat faster than the default cooldown
// while a condition persists (e.g. "low_fuel" re-fires every ~2 minutes in
// Game.log while fuel stays low — confirmed in sc_log_reader's research).
func (d *Deduplicator) SetTypeCooldown(eventType string, cooldown time.Duration) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.typeCooldowns[eventType] = cooldown
	if cooldown > d.maxCooldown {
		d.maxCooldown = cooldown
	}
}

// IsDuplicate returns true if this event was seen recently.
func (d *Deduplicator) IsDuplicate(evt Event) bool {
	key := eventKey(evt)
	cooldown := d.cooldown

	d.mu.Lock()
	defer d.mu.Unlock()

	if override, ok := d.typeCooldowns[evt.Type]; ok {
		cooldown = override
	}

	if lastSeen, ok := d.seen[key]; ok {
		if time.Since(lastSeen) < cooldown {
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
		cutoff := time.Now().Add(-d.maxCooldown * 2)
		for k, v := range d.seen {
			if v.Before(cutoff) {
				delete(d.seen, k)
			}
		}
		d.mu.Unlock()
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
