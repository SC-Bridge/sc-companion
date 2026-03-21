package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/SC-Bridge/sc-companion/internal/cigclient"
	"github.com/SC-Bridge/sc-companion/internal/config"
	"github.com/SC-Bridge/sc-companion/internal/events"
	"github.com/SC-Bridge/sc-companion/internal/logtailer"
	"github.com/SC-Bridge/sc-companion/internal/store"
	storesync "github.com/SC-Bridge/sc-companion/internal/sync"
	"github.com/SC-Bridge/sc-companion/internal/tray"
)

// App is the main application struct bound to the Wails frontend.
type App struct {
	ctx       context.Context
	cancel    context.CancelFunc
	cfg       *config.Config
	bus       *events.Bus
	db        *store.Store
	trayCtrl  *tray.Controller
	tailer    *logtailer.Tailer
	cigClient *cigclient.Client
	syncMgr   *cigclient.SyncManager
	mu        sync.Mutex
	debugMode bool

	// Recent events buffer for debug view
	eventsMu    sync.Mutex
	recentEvents []EventEntry
}

// EventEntry is a frontend-friendly event representation.
type EventEntry struct {
	Type      string            `json:"type"`
	Source    string            `json:"source"`
	Timestamp string            `json:"timestamp"`
	Data      map[string]string `json:"data"`
}

// StatusInfo represents the current app status for the frontend.
type StatusInfo struct {
	PlayerHandle  string `json:"playerHandle"`
	CurrentShip   string `json:"currentShip"`
	Location      string `json:"location"`
	Jurisdiction  string `json:"jurisdiction"`
	TailerActive  bool   `json:"tailerActive"`
	GameConnected bool   `json:"gameConnected"`
	SyncActive    bool   `json:"syncActive"`
	EventCount    int    `json:"eventCount"`
	LastEvent     string `json:"lastEvent"`
	DebugMode     bool   `json:"debugMode"`
}

// AppConfig represents the user-facing configuration.
type AppConfig struct {
	LogPath      string `json:"logPath"`
	APIEndpoint  string `json:"apiEndpoint"`
	APIToken     string `json:"apiToken"`
	DebugMode    bool   `json:"debugMode"`
}

const maxRecentEvents = 200

// NewApp creates the application struct.
func NewApp() *App {
	return &App{
		recentEvents: make([]EventEntry, 0, maxRecentEvents),
	}
}

// startup is called when the app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Load config
	cfg, err := config.Load("config.yaml")
	if err != nil {
		cfg = config.Default()
	}
	a.cfg = cfg

	// Open database
	dataDir := config.DataDir()
	os.MkdirAll(dataDir, 0700)
	dbPath := filepath.Join(dataDir, "companion.db")
	db, err := store.New(dbPath)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		return
	}
	a.db = db

	// Event bus
	a.bus = events.NewBus()

	// Dedup + coalesce
	dedup := events.NewDeduplicator(10 * time.Second)
	coalesce := events.NewCoalesceMultiLine()

	// Create a cancellable context for services
	svcCtx, svcCancel := context.WithCancel(context.Background())
	a.cancel = svcCancel

	// Tray controller
	a.trayCtrl = tray.NewController(a.bus, a.db, svcCancel)

	// Event subscriber — persist + buffer for debug
	a.bus.Subscribe(func(evt events.Event) {
		// Coalesce multi-line money events
		merged, emit := coalesce.Process(evt)
		if !emit {
			return
		}
		if merged.Type == "money_amount" {
			return
		}
		if dedup.IsDuplicate(merged) {
			return
		}

		// Persist
		if a.db != nil {
			if _, err := a.db.InsertEvent(merged); err != nil {
				slog.Error("store event failed", "type", merged.Type, "error", err)
			}
		}

		// Buffer for debug view
		entry := EventEntry{
			Type:      merged.Type,
			Source:    merged.Source,
			Timestamp: merged.Timestamp.Format("15:04:05.000"),
			Data:      merged.Data,
		}
		a.eventsMu.Lock()
		if len(a.recentEvents) >= maxRecentEvents {
			a.recentEvents = a.recentEvents[1:]
		}
		a.recentEvents = append(a.recentEvents, entry)
		a.eventsMu.Unlock()

		// Emit to frontend if debug mode
		if a.debugMode {
			runtime.EventsEmit(a.ctx, "event", entry)
		}

		slog.Debug("event", "type", merged.Type, "data", merged.Data)
	})

	// Start API sync
	if cfg.APIToken != "" {
		syncClient := storesync.NewClient(cfg.APIEndpoint, cfg.APIToken, db)
		go syncClient.Run(svcCtx)
		slog.Info("API sync enabled")
	}

	// Start log tailer
	logPath := cfg.LogPath
	if logPath == "" {
		logPath = config.DetectGameLog()
	}
	if logPath != "" {
		tailer, err := logtailer.New(logPath, a.bus)
		if err != nil {
			slog.Error("log tailer failed", "error", err)
		} else {
			a.tailer = tailer
			go tailer.Run(svcCtx)
			slog.Info("log tailer started", "path", logPath)
		}
	}

	// Watch for loginData.json and connect CIG client
	go func() {
		// Build list of all possible loginData.json paths
		var watchPaths []string
		drives := []string{"C", "D", "E", "F"}
		variants := []string{"LIVE", "PTU", "EPTU"}
		baseDirs := []string{
			"Roberts Space Industries\\StarCitizen",
			"Program Files\\Roberts Space Industries\\StarCitizen",
			"Games\\Roberts Space Industries\\StarCitizen",
		}
		for _, d := range drives {
			for _, base := range baseDirs {
				for _, v := range variants {
					p := filepath.Join(d+`:\`, base, v, "loginData.json")
					watchPaths = append(watchPaths, p)
				}
			}
		}

		slog.Info("watching for loginData.json", "paths", len(watchPaths))

		// Poll all paths for loginData.json
		cigclient.WatchLoginDataMulti(watchPaths, func(ld *cigclient.LoginData, path string) {
			slog.Info("loginData.json detected",
				"path", path,
				"username", ld.Username,
				"endpoint", ld.StarNetwork.ServicesEndpoint,
			)

			client, err := cigclient.NewClient(ld)
			if err != nil {
				slog.Error("failed to create CIG client", "error", err)
				return
			}

			if err := client.Connect(svcCtx); err != nil {
				slog.Error("failed to connect CIG client", "error", err)
				return
			}

			a.mu.Lock()
			if a.cigClient != nil {
				a.cigClient.Close()
			}
			a.cigClient = client
			a.mu.Unlock()

			slog.Info("CIG client connected", "username", ld.Username)

			// Emit connection event to bus for debug view
			a.bus.Publish(events.Event{
				Type:      "cig_connected",
				Source:    "grpc",
				Timestamp: time.Now(),
				Data:      map[string]string{"username": ld.Username},
			})

			// Start gRPC data sync manager (handles wallet, friends, rep, etc.)
			if cfg.APIToken != "" {
				syncMgr := cigclient.NewSyncManager(client, cfg.APIEndpoint, cfg.APIToken)
				syncMgr.SetOnSync(func(evt cigclient.SyncEvent) {
					data := map[string]string{
						"data_type": evt.DataType,
						"count":     fmt.Sprintf("%d", evt.Count),
					}
					evtType := "grpc_sync_ok"
					if evt.Error != "" {
						evtType = "grpc_sync_error"
						data["error"] = evt.Error
					}
					a.bus.Publish(events.Event{
						Type:      evtType,
						Source:    "grpc",
						Timestamp: time.Now(),
						Data:      data,
					})
				})
				a.mu.Lock()
				a.syncMgr = syncMgr
				a.mu.Unlock()
				go syncMgr.Run(svcCtx)
				slog.Info("gRPC sync manager started")
			} else {
				slog.Warn("gRPC sync manager not started — no API token configured")
			}
		})
	}()

	slog.Info("SC Bridge Companion started")
}

// shutdown is called when the app is closing.
func (a *App) shutdown(ctx context.Context) {
	if a.cancel != nil {
		a.cancel()
	}
	if a.cigClient != nil {
		a.cigClient.Close()
	}
	if a.db != nil {
		a.db.Close()
	}
	slog.Info("SC Bridge Companion stopped")
}

// --- Bound methods (called from React frontend) ---

// GetStatus returns the current application status.
func (a *App) GetStatus() StatusInfo {
	status := StatusInfo{
		DebugMode: a.debugMode,
	}

	if a.trayCtrl != nil {
		ts := a.trayCtrl.GetStatus()
		status.PlayerHandle = ts.PlayerHandle
		status.CurrentShip = ts.CurrentShip
		status.Location = ts.Location
		status.Jurisdiction = ts.Jurisdiction
		status.EventCount = ts.EventCount
		if !ts.LastEvent.IsZero() {
			status.LastEvent = ts.LastEvent.Format("15:04:05")
		}
	}

	status.TailerActive = a.tailer != nil

	a.mu.Lock()
	status.GameConnected = a.cigClient != nil
	status.SyncActive = a.syncMgr != nil
	a.mu.Unlock()

	return status
}

// GetConfig returns the current app configuration.
func (a *App) GetConfig() AppConfig {
	if a.cfg == nil {
		return AppConfig{}
	}
	return AppConfig{
		LogPath:     a.cfg.LogPath,
		APIEndpoint: a.cfg.APIEndpoint,
		APIToken:    a.cfg.APIToken,
		DebugMode:   a.debugMode,
	}
}

// SetDebugMode toggles debug mode (shows live event feed).
func (a *App) SetDebugMode(enabled bool) {
	a.debugMode = enabled
}

// GetRecentEvents returns buffered events for the debug view.
func (a *App) GetRecentEvents() []EventEntry {
	a.eventsMu.Lock()
	defer a.eventsMu.Unlock()
	out := make([]EventEntry, len(a.recentEvents))
	copy(out, a.recentEvents)
	return out
}

// GetEventCounts returns event type counts from the database.
func (a *App) GetEventCounts() map[string]int {
	if a.db == nil {
		return nil
	}
	counts, _ := a.db.EventCounts()
	return counts
}

// GetTotalEvents returns the total number of stored events.
func (a *App) GetTotalEvents() int {
	if a.db == nil {
		return 0
	}
	total, _ := a.db.TotalEvents()
	return total
}

// GetWallet returns the player's wallet balances from CIG's API.
func (a *App) GetWallet() ([]cigclient.WalletBalance, error) {
	a.mu.Lock()
	client := a.cigClient
	a.mu.Unlock()
	if client == nil {
		return nil, fmt.Errorf("not connected — launch Star Citizen first")
	}
	return client.GetWallet(context.Background())
}

// GetFriends returns the player's friend list from CIG's API.
func (a *App) GetFriends() ([]cigclient.Friend, error) {
	a.mu.Lock()
	client := a.cigClient
	a.mu.Unlock()
	if client == nil {
		return nil, fmt.Errorf("not connected — launch Star Citizen first")
	}
	return client.GetFriends(context.Background())
}

// GetReputation returns the player's reputation scores from CIG's API.
func (a *App) GetReputation() ([]cigclient.ReputationScore, error) {
	a.mu.Lock()
	client := a.cigClient
	a.mu.Unlock()
	if client == nil {
		return nil, fmt.Errorf("not connected — launch Star Citizen first")
	}
	return client.GetReputation(context.Background())
}

// GetBlueprints returns the player's blueprint collection from CIG's API.
func (a *App) GetBlueprints() ([]cigclient.Blueprint, error) {
	a.mu.Lock()
	client := a.cigClient
	a.mu.Unlock()
	if client == nil {
		return nil, fmt.Errorf("not connected — launch Star Citizen first")
	}
	return client.GetBlueprints(context.Background())
}

// GetEntitlements returns the player's entitlements from CIG's API.
func (a *App) GetEntitlements() ([]cigclient.Entitlement, error) {
	a.mu.Lock()
	client := a.cigClient
	a.mu.Unlock()
	if client == nil {
		return nil, fmt.Errorf("not connected — launch Star Citizen first")
	}
	return client.GetEntitlements(context.Background())
}

// GetActiveMissions returns the player's active missions from CIG's API.
func (a *App) GetActiveMissions() ([]cigclient.Mission, error) {
	a.mu.Lock()
	client := a.cigClient
	a.mu.Unlock()
	if client == nil {
		return nil, fmt.Errorf("not connected — launch Star Citizen first")
	}
	return client.GetActiveMissions(context.Background())
}

// GetStats returns the player's stats from CIG's API.
func (a *App) GetStats() ([]cigclient.PlayerStat, error) {
	a.mu.Lock()
	client := a.cigClient
	a.mu.Unlock()
	if client == nil {
		return nil, fmt.Errorf("not connected — launch Star Citizen first")
	}
	return client.GetStats(context.Background())
}

// IsGameConnected returns whether we have an active CIG API connection.
func (a *App) IsGameConnected() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.cigClient != nil
}

// TestAllGrpc fetches all gRPC data types and publishes results to the event bus.
// Called from the frontend debug UI to verify the CIG API is working.
func (a *App) TestAllGrpc() string {
	a.mu.Lock()
	client := a.cigClient
	a.mu.Unlock()
	if client == nil {
		return "not connected"
	}

	ctx := context.Background()
	var results []string

	// Wallet
	if wallets, err := client.GetWallet(ctx); err != nil {
		a.emitGrpcEvent("wallet_error", map[string]string{"error": err.Error()})
		results = append(results, "wallet: ERROR "+err.Error())
	} else {
		data := map[string]string{"count": fmt.Sprintf("%d", len(wallets))}
		for _, w := range wallets {
			data[w.Currency] = fmt.Sprintf("%d", w.Amount)
		}
		a.emitGrpcEvent("wallet_data", data)
		results = append(results, fmt.Sprintf("wallet: %d ledgers", len(wallets)))
	}

	// Friends
	if friends, err := client.GetFriends(ctx); err != nil {
		a.emitGrpcEvent("friends_error", map[string]string{"error": err.Error()})
		results = append(results, "friends: ERROR "+err.Error())
	} else {
		online := 0
		for _, f := range friends {
			if f.Status == "online" {
				online++
			}
		}
		a.emitGrpcEvent("friends_data", map[string]string{
			"total":  fmt.Sprintf("%d", len(friends)),
			"online": fmt.Sprintf("%d", online),
		})
		results = append(results, fmt.Sprintf("friends: %d total, %d online", len(friends), online))
	}

	// Reputation
	if scores, err := client.GetReputation(ctx); err != nil {
		a.emitGrpcEvent("reputation_error", map[string]string{"error": err.Error()})
		results = append(results, "reputation: ERROR "+err.Error())
	} else {
		a.emitGrpcEvent("reputation_data", map[string]string{
			"factions": fmt.Sprintf("%d", len(scores)),
		})
		results = append(results, fmt.Sprintf("reputation: %d factions", len(scores)))
	}

	// Blueprints
	if bps, err := client.GetBlueprints(ctx); err != nil {
		a.emitGrpcEvent("blueprints_error", map[string]string{"error": err.Error()})
		results = append(results, "blueprints: ERROR "+err.Error())
	} else {
		a.emitGrpcEvent("blueprints_data", map[string]string{
			"count": fmt.Sprintf("%d", len(bps)),
		})
		results = append(results, fmt.Sprintf("blueprints: %d", len(bps)))
	}

	// Entitlements
	if ents, err := client.GetEntitlements(ctx); err != nil {
		a.emitGrpcEvent("entitlements_error", map[string]string{"error": err.Error()})
		results = append(results, "entitlements: ERROR "+err.Error())
	} else {
		ships := 0
		for _, e := range ents {
			if e.ItemType == "SHIP" {
				ships++
			}
		}
		a.emitGrpcEvent("entitlements_data", map[string]string{
			"total": fmt.Sprintf("%d", len(ents)),
			"ships": fmt.Sprintf("%d", ships),
		})
		results = append(results, fmt.Sprintf("entitlements: %d total, %d ships", len(ents), ships))
	}

	// Missions
	if missions, err := client.GetActiveMissions(ctx); err != nil {
		a.emitGrpcEvent("missions_error", map[string]string{"error": err.Error()})
		results = append(results, "missions: ERROR "+err.Error())
	} else {
		a.emitGrpcEvent("missions_data", map[string]string{
			"count": fmt.Sprintf("%d", len(missions)),
		})
		results = append(results, fmt.Sprintf("missions: %d active", len(missions)))
	}

	// Stats
	if stats, err := client.GetStats(ctx); err != nil {
		a.emitGrpcEvent("stats_error", map[string]string{"error": err.Error()})
		results = append(results, "stats: ERROR "+err.Error())
	} else {
		a.emitGrpcEvent("stats_data", map[string]string{
			"count": fmt.Sprintf("%d", len(stats)),
		})
		results = append(results, fmt.Sprintf("stats: %d", len(stats)))
	}

	summary := strings.Join(results, " | ")
	slog.Info("TestAllGrpc complete", "summary", summary)
	return summary
}

func (a *App) emitGrpcEvent(eventType string, data map[string]string) {
	a.bus.Publish(events.Event{
		Type:      eventType,
		Source:    "grpc",
		Timestamp: time.Now(),
		Data:      data,
	})
}
