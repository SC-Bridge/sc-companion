package sync

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/SC-Bridge/sc-companion/internal/events"
	"github.com/SC-Bridge/sc-companion/internal/store"
)

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func insert(t *testing.T, s *store.Store, typ string) int64 {
	t.Helper()
	id, err := s.InsertEvent(events.Event{
		Type: typ, Source: "log", Timestamp: time.Now().UTC(),
		Data: map[string]string{"event_id": typ + "-id"},
	})
	if err != nil {
		t.Fatalf("insert %s: %v", typ, err)
	}
	return id
}

func pending(t *testing.T, s *store.Store) []store.StoredEvent {
	t.Helper()
	evs, err := s.UnsyncedEvents(100)
	if err != nil {
		t.Fatalf("unsynced: %v", err)
	}
	return evs
}

// captureServer records the events from each POST /companion/events body.
func captureServer(t *testing.T, status int, received *[]SyncPayload) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/companion/events", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var p SyncPayload
		_ = json.Unmarshal(body, &p)
		*received = append(*received, p)
		w.WriteHeader(status)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// A disabled event type must be marked skipped (recoverable), never sent, and
// the enabled one must be sent and marked synced.
func TestSyncBatch_SkipsDisabledWithoutLosingThem(t *testing.T) {
	s := newTestStore(t)
	insert(t, s, "fined")           // enabled
	insert(t, s, "location_change") // disabled

	var got []SyncPayload
	srv := captureServer(t, http.StatusOK, &got)

	c := NewClientWithAPIKey(srv.URL, "token", s)
	c.SetSyncCheck(func(typ string) bool { return typ == "fined" })

	if err := c.syncBatch(context.Background()); err != nil {
		t.Fatalf("syncBatch: %v", err)
	}

	// Server saw exactly the fined event, carrying its event_id.
	if len(got) != 1 || len(got[0].Events) != 1 || got[0].Events[0].Type != "fined" {
		t.Fatalf("expected only fined sent, got %+v", got)
	}
	if got[0].Events[0].Data["event_id"] != "fined-id" {
		t.Fatalf("expected event_id forwarded, got %q", got[0].Events[0].Data["event_id"])
	}

	// Nothing pending: fined is synced, location_change is skipped (not synced).
	if p := pending(t, s); len(p) != 0 {
		t.Fatalf("expected no pending events, got %v", p)
	}

	// Re-enabling location_change must recover it and let it sync.
	n, err := s.RequeueSkipped("location_change")
	if err != nil || n != 1 {
		t.Fatalf("requeue: n=%d err=%v", n, err)
	}
	got = nil
	c.SetSyncCheck(func(string) bool { return true })
	if err := c.syncBatch(context.Background()); err != nil {
		t.Fatalf("syncBatch after re-enable: %v", err)
	}
	if len(got) != 1 || len(got[0].Events) != 1 || got[0].Events[0].Type != "location_change" {
		t.Fatalf("expected location_change sent after re-enable, got %+v", got)
	}
}

// A POST failure must leave sendable events pending for retry (not lost).
func TestSyncBatch_PostFailureKeepsPending(t *testing.T) {
	s := newTestStore(t)
	insert(t, s, "fined")

	var got []SyncPayload
	srv := captureServer(t, http.StatusInternalServerError, &got)

	c := NewClientWithAPIKey(srv.URL, "token", s)
	c.SetSyncCheck(func(string) bool { return true })

	if err := c.syncBatch(context.Background()); err == nil {
		t.Fatal("expected error on 500 response")
	}
	if p := pending(t, s); len(p) != 1 {
		t.Fatalf("expected fined still pending after failure, got %v", p)
	}
}

// With no sendable events, skipped ones still advance out of the pending set so
// they don't block the batch window, while remaining recoverable.
func TestSyncBatch_AllDisabledMarksSkipped(t *testing.T) {
	s := newTestStore(t)
	insert(t, s, "location_change")

	var got []SyncPayload
	srv := captureServer(t, http.StatusOK, &got)

	c := NewClientWithAPIKey(srv.URL, "token", s)
	c.SetSyncCheck(func(string) bool { return false })

	if err := c.syncBatch(context.Background()); err != nil {
		t.Fatalf("syncBatch: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no POST when nothing sendable, got %+v", got)
	}
	if p := pending(t, s); len(p) != 0 {
		t.Fatalf("expected skipped event out of pending set, got %v", p)
	}
	if n, _ := s.RequeueSkipped("location_change"); n != 1 {
		t.Fatalf("expected skipped event recoverable, requeued %d", n)
	}
}
