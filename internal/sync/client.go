package sync

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/SC-Bridge/sc-companion/internal/store"
)

// Client syncs local events to the SC Bridge API.
type Client struct {
	endpoint   string
	authHeader string // "Bearer <token>" or legacy API token
	authValue  string
	httpClient *http.Client
	store      *store.Store
	batchSize  int
	syncCheck  func(string) bool // optional: returns true if event type should sync
	onAuthExpired func()         // optional: called on 401 response
}

// NewClient creates a sync client using Bearer token auth.
func NewClient(endpoint, sessionToken string, s *store.Store) *Client {
	return &Client{
		endpoint:   endpoint,
		authHeader: "Authorization",
		authValue:  "Bearer " + sessionToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		store:     s,
		batchSize: 100,
	}
}

// NewClientWithAPIKey creates a sync client using legacy X-API-Key auth.
func NewClientWithAPIKey(endpoint, apiToken string, s *store.Store) *Client {
	return &Client{
		endpoint:   endpoint,
		authHeader: "X-API-Key",
		authValue:  apiToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		store:     s,
		batchSize: 100,
	}
}

// SetSyncCheck sets a filter function for event types.
func (c *Client) SetSyncCheck(fn func(string) bool) {
	c.syncCheck = fn
}

// SetOnAuthExpired sets a callback for 401 responses.
func (c *Client) SetOnAuthExpired(fn func()) {
	c.onAuthExpired = fn
}

// SyncPayload is sent to the SC Bridge API.
type SyncPayload struct {
	Events []SyncEvent `json:"events"`
}

// SyncEvent is a single event in the sync payload.
type SyncEvent struct {
	Type      string            `json:"type"`
	Source    string            `json:"source"`
	Timestamp string            `json:"timestamp"`
	Data      map[string]string `json:"data"`
}

// Run starts the sync loop. Blocks until ctx is cancelled.
func (c *Client) Run(ctx context.Context) error {
	slog.Info("sync client started", "endpoint", c.endpoint)

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Final sync attempt before shutdown
			c.syncBatch(context.Background())
			return nil
		case <-ticker.C:
			if err := c.syncBatch(ctx); err != nil {
				slog.Error("sync failed", "error", err)
			}
		}
	}
}

func (c *Client) syncBatch(ctx context.Context) error {
	if c.authValue == "" {
		return nil // Not authenticated, skip sync
	}

	events, err := c.store.UnsyncedEvents(c.batchSize)
	if err != nil {
		return fmt.Errorf("fetch unsynced: %w", err)
	}

	if len(events) == 0 {
		return nil
	}

	// Filter events by sync preferences
	var filtered []store.StoredEvent
	for _, e := range events {
		if c.syncCheck != nil && !c.syncCheck(e.Type) {
			continue
		}
		filtered = append(filtered, e)
	}

	// Mark all as synced (even filtered-out ones — they shouldn't be retried)
	var maxID int64
	for _, e := range events {
		if e.ID > maxID {
			maxID = e.ID
		}
	}

	if len(filtered) == 0 {
		// Nothing to send, but still mark as synced
		if maxID > 0 {
			if err := c.store.MarkSynced(maxID); err != nil {
				return fmt.Errorf("mark synced: %w", err)
			}
		}
		return nil
	}

	payload := SyncPayload{
		Events: make([]SyncEvent, len(filtered)),
	}

	for i, e := range filtered {
		var data map[string]string
		if err := json.Unmarshal([]byte(e.DataJSON), &data); err != nil {
			data = map[string]string{"raw": e.DataJSON}
		}

		payload.Events[i] = SyncEvent{
			Type:      e.Type,
			Source:    e.Source,
			Timestamp: e.Timestamp,
			Data:      data,
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint+"/companion/events", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(c.authHeader, c.authValue)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode == http.StatusUnauthorized {
		slog.Warn("sync auth expired (401)")
		if c.onAuthExpired != nil {
			c.onAuthExpired()
		}
		return fmt.Errorf("auth expired (401)")
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("api returned %d", resp.StatusCode)
	}

	// Mark as synced
	if err := c.store.MarkSynced(maxID); err != nil {
		return fmt.Errorf("mark synced: %w", err)
	}

	slog.Info("synced events", "count", len(filtered), "max_id", maxID)
	return nil
}

// Heartbeat sends a status ping to SC Bridge with current state.
func (c *Client) Heartbeat(ctx context.Context, state map[string]string) error {
	if c.authValue == "" {
		return nil
	}

	body, _ := json.Marshal(state)
	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint+"/companion/heartbeat", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(c.authHeader, c.authValue)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode == http.StatusUnauthorized {
		slog.Warn("heartbeat auth expired (401)")
		if c.onAuthExpired != nil {
			c.onAuthExpired()
		}
	}

	return nil
}
