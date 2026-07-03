package events

import "github.com/google/uuid"

// EventIDKey is the Data map key under which every event carries its stable,
// unique identifier. The SC Bridge accountant bridge dedupes ledger entries on
// data.event_id, so two same-type events in the same log-timestamp second must
// carry distinct ids or the second is silently dropped from the ledger.
const EventIDKey = "event_id"

// StampID assigns a stable, unique event_id to the event's Data map, generated
// once at first persistence. It is a UUIDv4 — unique across the install's
// lifetime and, practically, across installs.
//
// The id must be byte-identical on every resend of the same event, so this is
// idempotent: if the event already carries an id (e.g. it was reloaded from the
// store) it is left untouched. Multi-line notifications are coalesced into a
// single Event upstream in the parser, so the merged event is stamped exactly
// once here rather than once per raw line.
func StampID(evt *Event) {
	if evt.Data == nil {
		evt.Data = make(map[string]string)
	}
	if _, ok := evt.Data[EventIDKey]; ok {
		return
	}
	evt.Data[EventIDKey] = uuid.NewString()
}
