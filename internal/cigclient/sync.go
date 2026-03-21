package cigclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// SyncEvent is emitted when a sync operation completes (for debug UI).
type SyncEvent struct {
	DataType string
	Count    int
	Error    string
}

// SyncManager polls CIG gRPC APIs on a schedule and POSTs data to scbridge.app.
type SyncManager struct {
	client     *Client
	endpoint   string
	apiToken   string
	httpClient *http.Client
	onSync     func(SyncEvent) // optional callback for debug events
}

// NewSyncManager creates a sync manager.
func NewSyncManager(client *Client, endpoint, apiToken string) *SyncManager {
	return &SyncManager{
		client:   client,
		endpoint: endpoint,
		apiToken: apiToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetOnSync registers a callback that fires after each sync attempt.
func (sm *SyncManager) SetOnSync(fn func(SyncEvent)) {
	sm.onSync = fn
}

func (sm *SyncManager) emitSync(dataType string, count int, err error) {
	if sm.onSync == nil {
		return
	}
	evt := SyncEvent{DataType: dataType, Count: count}
	if err != nil {
		evt.Error = err.Error()
	}
	sm.onSync(evt)
}

// Run starts all sync loops. Blocks until ctx is cancelled.
func (sm *SyncManager) Run(ctx context.Context) {
	slog.Info("sync manager started", "endpoint", sm.endpoint)

	// Stagger initial syncs to avoid thundering herd
	type syncTask struct {
		name     string
		interval time.Duration
		delay    time.Duration
		fn       func(context.Context)
	}

	tasks := []syncTask{
		{"wallet", 30 * time.Second, 0, sm.syncWallet},
		{"friends", 60 * time.Second, 5 * time.Second, sm.syncFriends},
		{"missions", 60 * time.Second, 10 * time.Second, sm.syncMissions},
		{"reputation", 5 * time.Minute, 15 * time.Second, sm.syncReputation},
		{"blueprints", 5 * time.Minute, 20 * time.Second, sm.syncBlueprints},
		{"stats", 10 * time.Minute, 25 * time.Second, sm.syncStats},
		{"entitlements", 10 * time.Minute, 30 * time.Second, sm.syncEntitlements},
	}

	for _, t := range tasks {
		go sm.runLoop(ctx, t.name, t.interval, t.delay, t.fn)
	}

	<-ctx.Done()
	slog.Info("sync manager stopped")
}

func (sm *SyncManager) runLoop(ctx context.Context, name string, interval, delay time.Duration, fn func(context.Context)) {
	// Initial delay for staggering
	if delay > 0 {
		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}
	}

	// Run immediately, then on ticker
	fn(ctx)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fn(ctx)
		}
	}
}

// postJSON sends data to a sync endpoint.
func (sm *SyncManager) postJSON(ctx context.Context, path string, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", sm.endpoint+"/companion/sync/"+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", sm.apiToken)

	resp, err := sm.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("api returned %d", resp.StatusCode)
	}

	return nil
}

func (sm *SyncManager) syncWallet(ctx context.Context) {
	wallets, err := sm.client.GetWallet(ctx)
	if err != nil {
		slog.Error("sync: wallet gRPC failed", "error", err)
		return
	}

	// Aggregate balances by currency
	var auec, uec, rec, mer uint64
	for _, w := range wallets {
		switch w.Currency {
		case "AUEC":
			auec += w.Amount
		case "UEC":
			uec += w.Amount
		case "REC":
			rec += w.Amount
		case "MER":
			mer += w.Amount
		}
	}

	payload := map[string]interface{}{
		"auec":        auec,
		"uec":         uec,
		"rec":         rec,
		"mer":         mer,
		"captured_at": time.Now().UTC().Format(time.RFC3339),
	}

	if err := sm.postJSON(ctx, "wallet", payload); err != nil {
		slog.Error("sync: wallet POST failed", "error", err)
		sm.emitSync("wallet", 0, err)
		return
	}
	slog.Debug("sync: wallet synced", "auec", auec)
	sm.emitSync("wallet", 1, nil)
}

func (sm *SyncManager) syncFriends(ctx context.Context) {
	friends, err := sm.client.GetFriends(ctx)
	if err != nil {
		slog.Error("sync: friends gRPC failed", "error", err)
		return
	}

	type friendPayload struct {
		AccountID   string `json:"account_id"`
		Nickname    string `json:"nickname,omitempty"`
		DisplayName string `json:"display_name,omitempty"`
		Presence    string `json:"presence"`
		Activity    string `json:"activity_state,omitempty"`
	}

	items := make([]friendPayload, len(friends))
	for i, f := range friends {
		items[i] = friendPayload{
			AccountID:   fmt.Sprintf("%d", f.AccountID),
			Nickname:    f.Nickname,
			DisplayName: f.DisplayName,
			Presence:    f.Status,
			Activity:    f.Activity,
		}
	}

	if err := sm.postJSON(ctx, "friends", map[string]interface{}{"friends": items}); err != nil {
		slog.Error("sync: friends POST failed", "error", err)
		sm.emitSync("friends", 0, err)
		return
	}
	slog.Debug("sync: friends synced", "count", len(friends))
	sm.emitSync("friends", len(friends), nil)
}

func (sm *SyncManager) syncReputation(ctx context.Context) {
	scores, err := sm.client.GetReputation(ctx)
	if err != nil {
		slog.Error("sync: reputation gRPC failed", "error", err)
		return
	}

	// Also fetch history for all entities
	var repIDs []string
	for _, s := range scores {
		repIDs = append(repIDs, s.EntityID)
	}

	var historyEntries []ReputationHistoryEntry
	if len(repIDs) > 0 {
		historyEntries, err = sm.client.GetReputationHistory(ctx, repIDs, 30)
		if err != nil {
			slog.Warn("sync: reputation history gRPC failed", "error", err)
			// Continue without history
		}
	}

	type histPayload struct {
		EntityID       string `json:"entity_id"`
		Scope          string `json:"scope"`
		Score          int    `json:"score"`
		EventTimestamp string `json:"event_timestamp"`
	}

	history := make([]histPayload, len(historyEntries))
	for i, h := range historyEntries {
		history[i] = histPayload{
			EntityID:       h.EntityID,
			Scope:          h.Scope,
			Score:          int(h.Score),
			EventTimestamp: time.Unix(int64(h.EventTimestamp), 0).UTC().Format(time.RFC3339),
		}
	}

	payload := map[string]interface{}{
		"scores":      scores,
		"history":     history,
		"captured_at": time.Now().UTC().Format(time.RFC3339),
	}

	if err := sm.postJSON(ctx, "reputation", payload); err != nil {
		slog.Error("sync: reputation POST failed", "error", err)
		sm.emitSync("reputation", 0, err)
		return
	}
	slog.Debug("sync: reputation synced", "scores", len(scores), "history", len(history))
	sm.emitSync("reputation", len(scores), nil)
}

func (sm *SyncManager) syncBlueprints(ctx context.Context) {
	blueprints, err := sm.client.GetBlueprints(ctx)
	if err != nil {
		slog.Error("sync: blueprints gRPC failed", "error", err)
		return
	}

	if err := sm.postJSON(ctx, "blueprints", map[string]interface{}{"blueprints": blueprints}); err != nil {
		slog.Error("sync: blueprints POST failed", "error", err)
		sm.emitSync("blueprints", 0, err)
		return
	}
	slog.Debug("sync: blueprints synced", "count", len(blueprints))
	sm.emitSync("blueprints", len(blueprints), nil)
}

func (sm *SyncManager) syncEntitlements(ctx context.Context) {
	entitlements, err := sm.client.GetEntitlements(ctx)
	if err != nil {
		slog.Error("sync: entitlements gRPC failed", "error", err)
		return
	}

	// Transform to API payload format
	type entPayload struct {
		URN               string `json:"urn"`
		Name              string `json:"name,omitempty"`
		EntityClassGUID   string `json:"entity_class_guid,omitempty"`
		EntitlementType   string `json:"entitlement_type"`
		Status            string `json:"status,omitempty"`
		ItemType          string `json:"item_type,omitempty"`
		Source            string `json:"source,omitempty"`
		InsuranceLifetime int    `json:"insurance_lifetime"`
		InsuranceDuration int    `json:"insurance_duration,omitempty"`
	}

	items := make([]entPayload, len(entitlements))
	for i, e := range entitlements {
		lifetime := 0
		if e.InsuranceLifetime {
			lifetime = 1
		}
		items[i] = entPayload{
			URN:               e.URN,
			Name:              e.Name,
			EntityClassGUID:   e.EntityClassGUID,
			EntitlementType:   e.EntitlementType,
			Status:            e.Status,
			ItemType:          e.ItemType,
			Source:            e.Source,
			InsuranceLifetime: lifetime,
			InsuranceDuration: int(e.InsuranceDuration),
		}
	}

	if err := sm.postJSON(ctx, "entitlements", map[string]interface{}{"entitlements": items}); err != nil {
		slog.Error("sync: entitlements POST failed", "error", err)
		sm.emitSync("entitlements", 0, err)
		return
	}
	slog.Debug("sync: entitlements synced", "count", len(entitlements))
	sm.emitSync("entitlements", len(entitlements), nil)
}

func (sm *SyncManager) syncMissions(ctx context.Context) {
	missions, err := sm.client.GetActiveMissions(ctx)
	if err != nil {
		slog.Error("sync: missions gRPC failed", "error", err)
		return
	}

	type missionPayload struct {
		MissionID      string `json:"mission_id"`
		ContractID     string `json:"contract_id,omitempty"`
		Template       string `json:"template,omitempty"`
		State          string `json:"state"`
		RewardAUEC     int    `json:"reward_auec,omitempty"`
		ExpiresAt      string `json:"expires_at,omitempty"`
		ObjectivesJSON string `json:"objectives_json,omitempty"`
	}

	items := make([]missionPayload, len(missions))
	for i, m := range missions {
		items[i] = missionPayload{
			MissionID:      m.MissionID,
			ContractID:     m.ContractID,
			Template:       m.Template,
			State:          m.State,
			RewardAUEC:     int(m.RewardAUEC),
			ExpiresAt:      m.ExpiresAt,
			ObjectivesJSON: m.Objectives,
		}
	}

	payload := map[string]interface{}{
		"missions":    items,
		"captured_at": time.Now().UTC().Format(time.RFC3339),
	}

	if err := sm.postJSON(ctx, "missions", payload); err != nil {
		slog.Error("sync: missions POST failed", "error", err)
		sm.emitSync("missions", 0, err)
		return
	}
	slog.Debug("sync: missions synced", "count", len(missions))
	sm.emitSync("missions", len(missions), nil)
}

func (sm *SyncManager) syncStats(ctx context.Context) {
	stats, err := sm.client.GetStats(ctx)
	if err != nil {
		slog.Error("sync: stats gRPC failed", "error", err)
		return
	}

	type statPayload struct {
		StatDefID string  `json:"stat_def_id"`
		Value     float64 `json:"value"`
		Best      float64 `json:"best,omitempty"`
		Category  string  `json:"category,omitempty"`
		GameMode  string  `json:"game_mode,omitempty"`
	}

	items := make([]statPayload, len(stats))
	for i, s := range stats {
		items[i] = statPayload{
			StatDefID: s.StatDefID,
			Value:     float64(s.Value),
			Best:      float64(s.Best),
			Category:  s.Category,
			GameMode:  s.GameMode,
		}
	}

	if err := sm.postJSON(ctx, "stats", map[string]interface{}{"stats": items}); err != nil {
		slog.Error("sync: stats POST failed", "error", err)
		sm.emitSync("stats", 0, err)
		return
	}
	slog.Debug("sync: stats synced", "count", len(stats))
	sm.emitSync("stats", len(stats), nil)
}
