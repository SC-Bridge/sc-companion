package store

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/SC-Bridge/sc-companion/internal/events"
	_ "modernc.org/sqlite"
)

// Sync state values stored in the events.synced column. The column has always
// been an INTEGER, so introducing SyncSkipped adds a value rather than a schema
// change — no migration required.
const (
	// SyncPending: not yet processed by the sync client.
	SyncPending = 0
	// SyncDone: successfully delivered to SC Bridge.
	SyncDone = 1
	// SyncSkipped: withheld because the event type is disabled in sync
	// preferences. Distinct from SyncDone so re-enabling the type can requeue
	// these for delivery — the server dedupes idempotently, so late delivery
	// of accountant-relevant events is safe.
	SyncSkipped = 2
)

// Store persists events to a local SQLite database.
type Store struct {
	db *sql.DB
}

// New opens (or creates) a SQLite database at the given path.
func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	// WAL mode for concurrent reads during writes
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("set WAL: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

// Close closes the database.
func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS events (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			type       TEXT    NOT NULL,
			source     TEXT    NOT NULL,
			timestamp  TEXT    NOT NULL,
			data_json  TEXT    NOT NULL,
			synced     INTEGER NOT NULL DEFAULT 0,
			created_at TEXT    NOT NULL DEFAULT (datetime('now'))
		);

		CREATE INDEX IF NOT EXISTS idx_events_type ON events(type);
		CREATE INDEX IF NOT EXISTS idx_events_synced ON events(synced) WHERE synced = 0;
		CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events(timestamp);

		CREATE TABLE IF NOT EXISTS state (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		);

		CREATE TABLE IF NOT EXISTS sync_cursor (
			id         INTEGER PRIMARY KEY CHECK (id = 1),
			last_event_id INTEGER NOT NULL DEFAULT 0,
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		);

		INSERT OR IGNORE INTO sync_cursor (id, last_event_id) VALUES (1, 0);
	`)
	return err
}

// InsertEvent stores a parsed event.
func (s *Store) InsertEvent(evt events.Event) (int64, error) {
	ts := evt.Timestamp.UTC().Format(time.RFC3339Nano)
	dataJSON := mapToJSON(evt.Data)

	result, err := s.db.Exec(
		"INSERT INTO events (type, source, timestamp, data_json) VALUES (?, ?, ?, ?)",
		evt.Type, evt.Source, ts, dataJSON,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// SetState upserts a key-value pair in the state table.
func (s *Store) SetState(key, value string) error {
	_, err := s.db.Exec(
		`INSERT INTO state (key, value, updated_at) VALUES (?, ?, datetime('now'))
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		key, value,
	)
	return err
}

// GetState retrieves a value from the state table.
func (s *Store) GetState(key string) (string, error) {
	var value string
	err := s.db.QueryRow("SELECT value FROM state WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

// UnsyncedEvents returns events that haven't been synced to SC Bridge yet.
func (s *Store) UnsyncedEvents(limit int) ([]StoredEvent, error) {
	rows, err := s.db.Query(
		"SELECT id, type, source, timestamp, data_json FROM events WHERE synced = 0 ORDER BY id LIMIT ?",
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []StoredEvent
	for rows.Next() {
		var e StoredEvent
		if err := rows.Scan(&e.ID, &e.Type, &e.Source, &e.Timestamp, &e.DataJSON); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// MarkSyncedIDs marks the given events as successfully delivered (SyncDone).
func (s *Store) MarkSyncedIDs(ids []int64) error {
	return s.setSyncState(ids, SyncDone)
}

// MarkSkippedIDs marks the given events as withheld by sync preferences
// (SyncSkipped). Unlike SyncDone, these remain requeueable via RequeueSkipped
// if the user later re-enables the type.
func (s *Store) MarkSkippedIDs(ids []int64) error {
	return s.setSyncState(ids, SyncSkipped)
}

// setSyncState updates the synced column for a set of event IDs in one
// statement. No-op for an empty slice.
func (s *Store) setSyncState(ids []int64, state int) error {
	if len(ids) == 0 {
		return nil
	}
	placeholders := make([]string, len(ids))
	args := make([]any, 0, len(ids)+1)
	args = append(args, state)
	for i, id := range ids {
		placeholders[i] = "?"
		args = append(args, id)
	}
	query := fmt.Sprintf("UPDATE events SET synced = ? WHERE id IN (%s)", strings.Join(placeholders, ","))
	_, err := s.db.Exec(query, args...)
	return err
}

// RequeueSkipped returns previously skipped events of a given type to the
// pending state so they become eligible for sync again. Called when the user
// re-enables a type whose events were withheld while it was disabled. Returns
// the number of events requeued.
func (s *Store) RequeueSkipped(eventType string) (int64, error) {
	res, err := s.db.Exec(
		"UPDATE events SET synced = ? WHERE type = ? AND synced = ?",
		SyncPending, eventType, SyncSkipped,
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// RequeueAllSkipped returns every skipped event to the pending state. Used when
// preferences are reset to defaults (which re-enable the accountant-critical
// economy types); the next sync pass re-skips any type still disabled. Returns
// the number of events requeued.
func (s *Store) RequeueAllSkipped() (int64, error) {
	res, err := s.db.Exec(
		"UPDATE events SET synced = ? WHERE synced = ?",
		SyncPending, SyncSkipped,
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// EventCounts returns the count of each event type.
func (s *Store) EventCounts() (map[string]int, error) {
	rows, err := s.db.Query("SELECT type, COUNT(*) FROM events GROUP BY type ORDER BY COUNT(*) DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var t string
		var c int
		if err := rows.Scan(&t, &c); err != nil {
			return nil, err
		}
		counts[t] = c
	}
	return counts, rows.Err()
}

// TotalEvents returns the total number of stored events.
func (s *Store) TotalEvents() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM events").Scan(&count)
	return count, err
}

// StoredEvent represents an event row from the database.
type StoredEvent struct {
	ID        int64
	Type      string
	Source    string
	Timestamp string
	DataJSON  string
}

// mapToJSON converts a map to a simple JSON string without importing encoding/json.
func mapToJSON(m map[string]string) string {
	if len(m) == 0 {
		return "{}"
	}
	result := "{"
	first := true
	for k, v := range m {
		if !first {
			result += ","
		}
		result += fmt.Sprintf("%q:%q", k, v)
		first = false
	}
	result += "}"
	return result
}

func init() {
	_ = slog.Default() // ensure slog is importable
}
