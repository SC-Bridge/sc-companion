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
	apiToken   string
	httpClient *http.Client
	store      *store.Store
	batchSize  int
}

// NewClient creates a sync client.
func NewClient(endpoint, apiToken string, s *store.Store) *Client {
	return &Client{
		endpoint: endpoint,
		apiToken: apiToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		store:     s,
		batchSize: 100,
	}
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
	if c.apiToken == "" {
		return nil // Not authenticated, skip sync
	}

	events, err := c.store.UnsyncedEvents(c.batchSize)
	if err != nil {
		return fmt.Errorf("fetch unsynced: %w", err)
	}

	if len(events) == 0 {
		return nil
	}

	payload := SyncPayload{
		Events: make([]SyncEvent, len(events)),
	}

	var maxID int64
	for i, e := range events {
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

		if e.ID > maxID {
			maxID = e.ID
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
	req.Header.Set("X-API-Key", c.apiToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("api returned %d", resp.StatusCode)
	}

	// Mark as synced
	if err := c.store.MarkSynced(maxID); err != nil {
		return fmt.Errorf("mark synced: %w", err)
	}

	slog.Info("synced events", "count", len(events), "max_id", maxID)
	return nil
}

// Heartbeat sends a status ping to SC Bridge with current state.
func (c *Client) Heartbeat(ctx context.Context, state map[string]string) error {
	if c.apiToken == "" {
		return nil
	}

	body, _ := json.Marshal(state)
	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint+"/companion/heartbeat", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.apiToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	return nil
}
