package events

import "testing"

func TestStampID_AssignsUniqueID(t *testing.T) {
	a := Event{Type: "fined", Data: map[string]string{"amount": "500"}}
	b := Event{Type: "fined", Data: map[string]string{"amount": "500"}}

	StampID(&a)
	StampID(&b)

	if a.Data[EventIDKey] == "" {
		t.Fatal("expected event_id to be set")
	}
	// Two same-type, same-second events must get distinct ids, or the server
	// bridge drops the second on its composite fallback key.
	if a.Data[EventIDKey] == b.Data[EventIDKey] {
		t.Fatalf("expected distinct ids, both were %q", a.Data[EventIDKey])
	}
	if got := len(a.Data[EventIDKey]); got != 36 {
		t.Fatalf("expected 36-char UUIDv4, got %d chars: %q", got, a.Data[EventIDKey])
	}
}

func TestStampID_Idempotent(t *testing.T) {
	// A resend of a stored event must carry the byte-identical id, so stamping
	// an event that already has one must not regenerate it.
	evt := Event{Type: "fined", Data: map[string]string{EventIDKey: "preset-id"}}
	StampID(&evt)
	if evt.Data[EventIDKey] != "preset-id" {
		t.Fatalf("expected id preserved, got %q", evt.Data[EventIDKey])
	}
}

func TestStampID_NilData(t *testing.T) {
	evt := Event{Type: "qt_arrived"}
	StampID(&evt)
	if evt.Data[EventIDKey] == "" {
		t.Fatal("expected event_id on event with nil Data map")
	}
}
