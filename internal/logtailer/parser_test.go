package logtailer

import (
	"testing"

	"github.com/SC-Bridge/sc-companion/internal/events"
)

// A multi-line money_sent notification must coalesce into a single event so it
// receives exactly one event_id downstream — never one per raw line.
func TestParser_MoneySentCoalesces(t *testing.T) {
	p := NewParser()

	// Line 1: the "You sent <recipient>:" header — buffered, emits nothing.
	if _, ok := p.Parse(`<2026-07-03T18:30:55.361Z> [Notice] Added notification "You sent Vengeance: `); ok {
		t.Fatal("header line should not emit an event on its own")
	}

	// Line 2: the amount continuation — emits the merged event.
	evt, ok := p.Parse(`<2026-07-03T18:30:56.000Z> 7035000 aUEC`)
	if !ok {
		t.Fatal("amount line should emit the merged money_sent event")
	}
	if evt.Type != "money_sent" {
		t.Fatalf("expected type money_sent, got %q", evt.Type)
	}
	if evt.Data["recipient"] != "Vengeance" {
		t.Fatalf("expected recipient Vengeance, got %q", evt.Data["recipient"])
	}
	if evt.Data["amount"] != "7035000" {
		t.Fatalf("expected amount 7035000, got %q", evt.Data["amount"])
	}

	// The single merged event gets exactly one id.
	events.StampID(&evt)
	if evt.Data["event_id"] == "" {
		t.Fatal("expected the merged event to carry one event_id")
	}
}

// Two fines in the same log-timestamp second are distinct economy events; each
// must get its own event_id so the accountant bridge keeps both.
func TestParser_TwoFinesGetDistinctIDs(t *testing.T) {
	p := NewParser()

	e1, ok1 := p.Parse(`<2026-07-03T18:30:55.000Z> [Notice] Added notification "Fined 500 UEC`)
	e2, ok2 := p.Parse(`<2026-07-03T18:30:55.000Z> [Notice] Added notification "Fined 750 UEC`)
	if !ok1 || !ok2 {
		t.Fatalf("expected both fines to parse, got ok1=%v ok2=%v", ok1, ok2)
	}

	events.StampID(&e1)
	events.StampID(&e2)
	if e1.Data["event_id"] == e2.Data["event_id"] {
		t.Fatalf("expected distinct ids for two fines, both %q", e1.Data["event_id"])
	}
}
