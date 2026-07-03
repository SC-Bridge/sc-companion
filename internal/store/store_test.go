package store

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/SC-Bridge/sc-companion/internal/events"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func insert(t *testing.T, s *Store, typ string, data map[string]string) int64 {
	t.Helper()
	id, err := s.InsertEvent(events.Event{
		Type:      typ,
		Source:    "log",
		Timestamp: time.Now().UTC(),
		Data:      data,
	})
	if err != nil {
		t.Fatalf("insert %s: %v", typ, err)
	}
	return id
}

func unsyncedIDs(t *testing.T, s *Store) []int64 {
	t.Helper()
	evs, err := s.UnsyncedEvents(100)
	if err != nil {
		t.Fatalf("unsynced: %v", err)
	}
	ids := make([]int64, len(evs))
	for i, e := range evs {
		ids[i] = e.ID
	}
	return ids
}

// The event_id stamped before persistence must survive the store round-trip
// byte-for-byte — server idempotency on resend depends on it.
func TestEventIDRoundTrip(t *testing.T) {
	s := newTestStore(t)
	insert(t, s, "fined", map[string]string{"amount": "500", "event_id": "abc-123"})

	evs, err := s.UnsyncedEvents(100)
	if err != nil {
		t.Fatalf("unsynced: %v", err)
	}
	if len(evs) != 1 {
		t.Fatalf("expected 1 event, got %d", len(evs))
	}
	var data map[string]string
	if err := json.Unmarshal([]byte(evs[0].DataJSON), &data); err != nil {
		t.Fatalf("unmarshal data_json: %v", err)
	}
	if data["event_id"] != "abc-123" {
		t.Fatalf("expected event_id preserved, got %q", data["event_id"])
	}
}

func TestMarkSyncedIDs(t *testing.T) {
	s := newTestStore(t)
	id1 := insert(t, s, "fined", nil)
	id2 := insert(t, s, "money_sent", nil)

	if err := s.MarkSyncedIDs([]int64{id1}); err != nil {
		t.Fatalf("mark synced: %v", err)
	}
	got := unsyncedIDs(t, s)
	if len(got) != 1 || got[0] != id2 {
		t.Fatalf("expected only id2 pending, got %v", got)
	}
}

// A skipped event must not be treated as synced, and must be recoverable when
// its type is re-enabled — this is the core data-loss fix for the accountant.
func TestMarkSkippedThenRequeue(t *testing.T) {
	s := newTestStore(t)
	id1 := insert(t, s, "fined", nil)
	id2 := insert(t, s, "location_change", nil)

	if err := s.MarkSkippedIDs([]int64{id2}); err != nil {
		t.Fatalf("mark skipped: %v", err)
	}
	// Skipped event drops out of the pending set...
	if got := unsyncedIDs(t, s); len(got) != 1 || got[0] != id1 {
		t.Fatalf("expected only id1 pending after skip, got %v", got)
	}

	// ...but re-enabling its type brings it back, eligible for sync.
	n, err := s.RequeueSkipped("location_change")
	if err != nil {
		t.Fatalf("requeue: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 requeued, got %d", n)
	}
	got := unsyncedIDs(t, s)
	if len(got) != 2 || got[0] != id1 || got[1] != id2 {
		t.Fatalf("expected id1,id2 pending after requeue, got %v", got)
	}
}

// RequeueSkipped must only touch the named type and only skipped rows.
func TestRequeueSkipped_Scoped(t *testing.T) {
	s := newTestStore(t)
	finedID := insert(t, s, "fined", nil)
	locID := insert(t, s, "location_change", nil)
	syncedID := insert(t, s, "fined", nil)

	if err := s.MarkSkippedIDs([]int64{finedID, locID}); err != nil {
		t.Fatalf("skip: %v", err)
	}
	if err := s.MarkSyncedIDs([]int64{syncedID}); err != nil {
		t.Fatalf("sync: %v", err)
	}

	n, err := s.RequeueSkipped("fined")
	if err != nil {
		t.Fatalf("requeue: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected only the skipped fined event requeued, got %d", n)
	}
	// Only the fined skipped event returns; location_change stays skipped,
	// the already-synced fined event stays synced.
	if got := unsyncedIDs(t, s); len(got) != 1 || got[0] != finedID {
		t.Fatalf("expected only finedID pending, got %v", got)
	}
}

func TestRequeueAllSkipped(t *testing.T) {
	s := newTestStore(t)
	a := insert(t, s, "fined", nil)
	b := insert(t, s, "location_change", nil)
	if err := s.MarkSkippedIDs([]int64{a, b}); err != nil {
		t.Fatalf("skip: %v", err)
	}

	n, err := s.RequeueAllSkipped()
	if err != nil {
		t.Fatalf("requeue all: %v", err)
	}
	if n != 2 {
		t.Fatalf("expected 2 requeued, got %d", n)
	}
	if got := unsyncedIDs(t, s); len(got) != 2 {
		t.Fatalf("expected both pending after requeue-all, got %v", got)
	}
}

func TestSetSyncState_EmptyNoop(t *testing.T) {
	s := newTestStore(t)
	insert(t, s, "fined", nil)
	if err := s.MarkSyncedIDs(nil); err != nil {
		t.Fatalf("empty mark synced should be no-op, got %v", err)
	}
	if err := s.MarkSkippedIDs([]int64{}); err != nil {
		t.Fatalf("empty mark skipped should be no-op, got %v", err)
	}
	if got := unsyncedIDs(t, s); len(got) != 1 {
		t.Fatalf("expected event still pending, got %v", got)
	}
}
