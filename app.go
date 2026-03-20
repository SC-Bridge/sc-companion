package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/SC-Bridge/sc-companion/internal/config"
	"github.com/SC-Bridge/sc-companion/internal/events"
	"github.com/SC-Bridge/sc-companion/internal/grpcproxy"
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
	proxy     *grpcproxy.Proxy
	tailer    *logtailer.Tailer
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
	PlayerHandle string `json:"playerHandle"`
	CurrentShip  string `json:"currentShip"`
	Location     string `json:"location"`
	Jurisdiction string `json:"jurisdiction"`
	ProxyRunning bool   `json:"proxyRunning"`
	TailerActive bool   `json:"tailerActive"`
	EventCount   int    `json:"eventCount"`
	LastEvent    string `json:"lastEvent"`
	DebugMode    bool   `json:"debugMode"`
}

// AppConfig represents the user-facing configuration.
type AppConfig struct {
	LogPath      string `json:"logPath"`
	APIEndpoint  string `json:"apiEndpoint"`
	APIToken     string `json:"apiToken"`
	ProxyEnabled bool   `json:"proxyEnabled"`
	ProxyPort    int    `json:"proxyPort"`
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

	// Start gRPC proxy
	if cfg.ProxyEnabled {
		proxy, err := grpcproxy.NewProxy(grpcproxy.ProxyConfig{
			ListenAddr: fmt.Sprintf("127.0.0.1:%d", cfg.ProxyPort),
			CADir:      config.DataDir(),
		}, a.bus)
		if err != nil {
			slog.Error("gRPC proxy failed to start", "error", err)
		} else {
			a.proxy = proxy
			go proxy.Run(svcCtx)
		}
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

	slog.Info("SC Bridge Companion started")
}

// shutdown is called when the app is closing.
func (a *App) shutdown(ctx context.Context) {
	if a.cancel != nil {
		a.cancel()
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

	if a.proxy != nil {
		status.ProxyRunning = a.proxy.IsRunning()
	}

	status.TailerActive = a.tailer != nil

	return status
}

// GetConfig returns the current app configuration.
func (a *App) GetConfig() AppConfig {
	if a.cfg == nil {
		return AppConfig{}
	}
	return AppConfig{
		LogPath:      a.cfg.LogPath,
		APIEndpoint:  a.cfg.APIEndpoint,
		APIToken:     a.cfg.APIToken,
		ProxyEnabled: a.cfg.ProxyEnabled,
		ProxyPort:    a.cfg.ProxyPort,
		DebugMode:    a.debugMode,
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
