package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"sync"
	"time"

	"fyne.io/systray"
	wailsrt "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/SC-Bridge/sc-companion/internal/auth"
	"github.com/SC-Bridge/sc-companion/internal/updater"
	"github.com/SC-Bridge/sc-companion/internal/config"
	"github.com/SC-Bridge/sc-companion/internal/events"
	"github.com/SC-Bridge/sc-companion/internal/logtailer"
	"github.com/SC-Bridge/sc-companion/internal/store"
	storesync "github.com/SC-Bridge/sc-companion/internal/sync"
	"github.com/SC-Bridge/sc-companion/internal/tray"
)

// App is the main application struct bound to the Wails frontend.
type App struct {
	ctx          context.Context
	cancel       context.CancelFunc
	svcCtx       context.Context
	cfg          *config.Config
	cfgPath      string
	bus          *events.Bus
	db           *store.Store
	trayCtrl     *tray.Controller
	tailer       *logtailer.Tailer
	tailerCancel context.CancelFunc
	mu           sync.Mutex

	// Auth
	authInfo   *auth.AuthInfo
	syncClient *storesync.Client
	syncCancel context.CancelFunc

	// Sync preferences
	syncPrefs *config.SyncPreferences

	// Event log file (JSONL for WingmanAI)
	eventLog *os.File

	// Recent events buffer
	eventsMu     sync.Mutex
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
	TailerActive bool   `json:"tailerActive"`
	EventCount   int    `json:"eventCount"`
	LastEvent    string `json:"lastEvent"`
	Connected    bool   `json:"connected"`
	Handle       string `json:"handle"`
	Environment  string `json:"environment"`
}

// AppConfig represents the user-facing configuration.
type AppConfig struct {
	LogPath     string `json:"logPath"`
	APIEndpoint string `json:"apiEndpoint"`
	Environment string `json:"environment"`
	Connected   bool   `json:"connected"`
	Handle      string `json:"handle"`
}

// ConnectionStatus represents the auth connection state for the frontend.
type ConnectionStatus struct {
	Connected   bool   `json:"connected"`
	Handle      string `json:"handle"`
	Endpoint    string `json:"endpoint"`
	ConnectedAt string `json:"connectedAt"`
}

// Version is set at build time via -ldflags.
var Version = "0.3.12"

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
	a.cfgPath = "config.yaml"
	cfg, err := config.Load(a.cfgPath)
	if err != nil {
		cfg = config.Default()
	}
	a.cfg = cfg

	// Load sync preferences
	a.syncPrefs = config.LoadSyncPreferences()

	// Load auth
	a.authInfo = auth.Load()

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

	// Event log file (JSONL — one JSON object per line, for WingmanAI)
	eventLogPath := filepath.Join(dataDir, "events.log")
	eventLog, err := os.OpenFile(eventLogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		slog.Error("failed to open event log", "path", eventLogPath, "error", err)
	} else {
		a.eventLog = eventLog
		slog.Info("event log opened", "path", eventLogPath)
	}

	// Event bus
	a.bus = events.NewBus()

	// Dedup + coalesce
	dedup := events.NewDeduplicator(10 * time.Second)
	coalesce := events.NewCoalesceMultiLine()

	// Create a cancellable context for services
	svcCtx, svcCancel := context.WithCancel(context.Background())
	a.cancel = svcCancel
	a.svcCtx = svcCtx

	// Tray controller
	a.trayCtrl = tray.NewController(a.bus, a.db, svcCancel)

	// Event subscriber — persist + buffer + always emit to frontend
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

		// Persist to SQLite
		if a.db != nil {
			if _, err := a.db.InsertEvent(merged); err != nil {
				slog.Error("store event failed", "type", merged.Type, "error", err)
			}
		}

		// Write to JSONL event log
		if a.eventLog != nil {
			logEntry := map[string]interface{}{
				"type":      merged.Type,
				"source":    merged.Source,
				"timestamp": merged.Timestamp.Format(time.RFC3339Nano),
				"data":      merged.Data,
			}
			if line, err := json.Marshal(logEntry); err == nil {
				line = append(line, '\n')
				a.eventLog.Write(line)
			}
		}

		// Buffer for event feed
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

		// Always emit to frontend
		wailsrt.EventsEmit(a.ctx, "event", entry)

		slog.Debug("event", "type", merged.Type, "data", merged.Data)
	})

	// Start API sync if authenticated
	a.startSync(svcCtx)

	// Legacy API token fallback
	if a.syncClient == nil && cfg.APIToken != "" {
		slog.Warn("using legacy api_token from config.yaml — please use Connect to SC Bridge instead")
		endpoint := config.EndpointForEnv(cfg.Environment)
		client := storesync.NewClientWithAPIKey(endpoint, cfg.APIToken, db)
		client.SetSyncCheck(a.syncPrefs.IsEnabled)
		a.syncClient = client
		go client.Run(svcCtx)
		slog.Info("API sync enabled (legacy token)")
	}

	// Start log tailer
	logPath := cfg.LogPath
	if logPath == "" {
		logPath = config.DetectGameLog()
	}
	a.restartTailer(logPath)

	// System tray
	go systray.Run(a.onSystrayReady, nil)

	// Minimize-to-tray watcher
	go a.minimizeWatcher(svcCtx)

	slog.Info("SC Bridge Companion started")
}

// startSync starts the sync client if auth is available.
func (a *App) startSync(ctx context.Context) {
	if a.authInfo == nil || a.authInfo.SessionToken == "" || a.db == nil {
		return
	}

	endpoint := config.EndpointForEnv(a.cfg.Environment)
	client := storesync.NewClient(endpoint, a.authInfo.SessionToken, a.db)
	client.SetSyncCheck(a.syncPrefs.IsEnabled)
	client.SetOnAuthExpired(func() {
		wailsrt.EventsEmit(a.ctx, "auth_expired", nil)
	})

	syncCtx, syncCancel := context.WithCancel(ctx)
	a.syncClient = client
	a.syncCancel = syncCancel
	go client.Run(syncCtx)
	slog.Info("API sync enabled (bearer token)")
}

// shutdown is called when the app is closing.
func (a *App) shutdown(ctx context.Context) {
	if a.cancel != nil {
		a.cancel()
	}
	if a.eventLog != nil {
		a.eventLog.Close()
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
		Environment: a.cfg.Environment,
		Connected:   a.authInfo != nil,
	}

	if a.authInfo != nil {
		status.Handle = a.authInfo.Handle
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

	return status
}

// GetConfig returns the current app configuration.
func (a *App) GetConfig() AppConfig {
	if a.cfg == nil {
		return AppConfig{}
	}
	cfg := AppConfig{
		LogPath:     a.cfg.LogPath,
		APIEndpoint: config.EndpointForEnv(a.cfg.Environment),
		Environment: a.cfg.Environment,
		Connected:   a.authInfo != nil,
	}
	if a.authInfo != nil {
		cfg.Handle = a.authInfo.Handle
	}
	return cfg
}

// GetRecentEvents returns buffered events for the event feed.
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

// GetEventLogPath returns the path to the JSONL event log file.
func (a *App) GetEventLogPath() string {
	return filepath.Join(config.DataDir(), "events.log")
}

// GetDatabasePath returns the path to the SQLite database.
func (a *App) GetDatabasePath() string {
	return filepath.Join(config.DataDir(), "companion.db")
}

// GetDataDir returns the application data directory.
func (a *App) GetDataDir() string {
	return config.DataDir()
}

// OpenInExplorer opens the containing folder of a file path in the OS file manager.
func (a *App) OpenInExplorer(filePath string) {
	dir := filepath.Dir(filePath)
	if _, err := os.Stat(dir); err != nil {
		slog.Error("directory not found", "path", dir)
		return
	}
	var cmd *exec.Cmd
	switch goruntime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", "/select,", filePath)
	case "darwin":
		cmd = exec.Command("open", "-R", filePath)
	default:
		cmd = exec.Command("xdg-open", dir)
	}
	if err := cmd.Start(); err != nil {
		slog.Error("failed to open explorer", "error", err)
	}
}

// BrowseGameLog opens a file dialog to select the Game.log path, saves it,
// and (re)starts the log tailer against the chosen file.
func (a *App) BrowseGameLog() string {
	selection, err := wailsrt.OpenFileDialog(a.ctx, wailsrt.OpenDialogOptions{
		Title: "Select Game.log",
		Filters: []wailsrt.FileFilter{
			{DisplayName: "Log Files (*.log)", Pattern: "*.log"},
			{DisplayName: "All Files (*.*)", Pattern: "*.*"},
		},
	})
	if err != nil || selection == "" {
		return ""
	}

	a.mu.Lock()
	a.cfg.LogPath = selection
	a.cfg.Save(a.cfgPath)
	a.mu.Unlock()

	a.restartTailer(selection)
	return selection
}

// restartTailer stops any running tailer and starts a new one for the given path.
// A no-op if path is empty.
func (a *App) restartTailer(path string) {
	if path == "" {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()

	// Stop existing tailer
	if a.tailerCancel != nil {
		a.tailerCancel()
		a.tailerCancel = nil
		a.tailer = nil
	}

	tailer, err := logtailer.New(path, a.bus)
	if err != nil {
		slog.Error("log tailer failed", "path", path, "error", err)
		return
	}

	ctx, cancel := context.WithCancel(a.svcCtx)
	a.tailer = tailer
	a.tailerCancel = cancel
	go tailer.Run(ctx)
	slog.Info("log tailer started", "path", path)
}

// --- Environment switcher ---

// GetEnvironment returns the current environment.
func (a *App) GetEnvironment() string {
	return a.cfg.Environment
}

// SetEnvironment changes the environment and persists to config.
func (a *App) SetEnvironment(env string) error {
	if env != "production" && env != "staging" {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()

	a.cfg.Environment = env
	if err := a.cfg.Save(a.cfgPath); err != nil {
		slog.Error("failed to save config", "error", err)
		return err
	}
	slog.Info("environment changed", "env", env)
	return nil
}

// --- Sync preferences ---

// GetEventCategories returns all event types grouped by category.
func (a *App) GetEventCategories() []events.EventCategory {
	return events.EventCategories()
}

// GetSyncPreferences returns the current sync preferences.
func (a *App) GetSyncPreferences() map[string]bool {
	return a.syncPrefs.SyncEnabled
}

// SetSyncPreference updates a single event type's sync preference.
func (a *App) SetSyncPreference(eventType string, enabled bool) {
	a.syncPrefs.SyncEnabled[eventType] = enabled
	if err := a.syncPrefs.Save(); err != nil {
		slog.Error("failed to save sync preferences", "error", err)
	}
}

// ResetSyncPreferences resets all sync preferences to defaults.
func (a *App) ResetSyncPreferences() map[string]bool {
	a.syncPrefs = config.DefaultSyncPreferences()
	if err := a.syncPrefs.Save(); err != nil {
		slog.Error("failed to save sync preferences", "error", err)
	}
	return a.syncPrefs.SyncEnabled
}

// --- Friends list ---

// FriendEntry represents a friend from SC Bridge.
type FriendEntry struct {
	AccountID      string `json:"account_id"`
	Nickname       string `json:"nickname"`
	DisplayName    string `json:"display_name"`
	Presence       string `json:"presence"`
	ActivityState  string `json:"activity_state"`
	ActivityDetail string `json:"activity_detail"`
	UpdatedAt      string `json:"updated_at"`
}

type friendsResponse struct {
	OK      bool          `json:"ok"`
	Friends []FriendEntry `json:"friends"`
}

// GetFriends fetches the full friends list from SC Bridge.
func (a *App) GetFriends() []FriendEntry {
	return a.fetchFriends("")
}

// GetFriendsDelta fetches only friends updated since the given ISO timestamp.
func (a *App) GetFriendsDelta(since string) []FriendEntry {
	return a.fetchFriends(since)
}

func (a *App) fetchFriends(since string) []FriendEntry {
	if a.authInfo == nil || a.authInfo.SessionToken == "" {
		return nil
	}

	endpoint := config.EndpointForEnv(a.cfg.Environment)
	url := endpoint + "/companion/friends"
	if since != "" {
		url += "?since=" + since
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		slog.Error("friends request failed", "error", err)
		return nil
	}
	req.Header.Set("Authorization", "Bearer "+a.authInfo.SessionToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Error("friends fetch failed", "error", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		slog.Warn("friends fetch non-200", "status", resp.StatusCode)
		return nil
	}

	var result friendsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		slog.Error("friends decode failed", "error", err)
		return nil
	}

	return result.Friends
}

// --- OAuth connection ---

// ConnectToSCBridge starts the OAuth flow to connect to SC Bridge.
func (a *App) ConnectToSCBridge() ConnectionStatus {
	endpoint := config.EndpointForEnv(a.cfg.Environment)

	flow, err := auth.NewOAuthFlow(endpoint)
	if err != nil {
		slog.Error("oauth flow setup failed", "error", err)
		return ConnectionStatus{}
	}

	// Open browser
	connectURL := flow.ConnectURL()
	wailsrt.BrowserOpenURL(a.ctx, connectURL)

	// Wait for callback (blocks up to 5 min)
	result := flow.Start(a.ctx)
	if result.Error != nil {
		slog.Error("oauth flow failed", "error", result.Error)
		return ConnectionStatus{}
	}

	// Save auth info (handle will be populated on first sync/heartbeat)
	info := auth.NewAuthInfo(result.Token, "", endpoint)
	if err := auth.Save(info); err != nil {
		slog.Error("failed to save auth", "error", err)
		return ConnectionStatus{}
	}

	a.mu.Lock()
	a.authInfo = info
	a.mu.Unlock()

	// Start sync with new token
	if a.syncCancel != nil {
		a.syncCancel()
	}
	svcCtx, svcCancel := context.WithCancel(context.Background())
	a.cancel = svcCancel
	a.startSync(svcCtx)

	slog.Info("connected to SC Bridge")
	return ConnectionStatus{
		Connected:   true,
		Handle:      info.Handle,
		Endpoint:    info.Endpoint,
		ConnectedAt: info.ConnectedAt,
	}
}

// DisconnectFromSCBridge clears the auth and stops sync.
func (a *App) DisconnectFromSCBridge() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.syncCancel != nil {
		a.syncCancel()
		a.syncCancel = nil
	}
	a.syncClient = nil
	a.authInfo = nil

	if err := auth.Clear(); err != nil {
		slog.Error("failed to clear auth", "error", err)
	}
	slog.Info("disconnected from SC Bridge")
}

// GetConnectionStatus returns the current connection state.
func (a *App) GetConnectionStatus() ConnectionStatus {
	if a.authInfo == nil {
		return ConnectionStatus{}
	}
	return ConnectionStatus{
		Connected:   true,
		Handle:      a.authInfo.Handle,
		Endpoint:    a.authInfo.Endpoint,
		ConnectedAt: a.authInfo.ConnectedAt,
	}
}

// --- Version & updates ---

// GetVersion returns the current app version.
func (a *App) GetVersion() string {
	return Version
}

// CheckForUpdate checks GitHub Releases for a newer version.
func (a *App) CheckForUpdate() *updater.ReleaseInfo {
	info, err := updater.CheckForUpdate(Version)
	if err != nil {
		slog.Error("update check failed", "error", err)
		return &updater.ReleaseInfo{Version: Version, HasUpdate: false}
	}
	return info
}

// OpenDownloadURL opens the download URL in the default browser.
func (a *App) OpenDownloadURL(url string) {
	wailsrt.BrowserOpenURL(a.ctx, url)
}

// ApplyUpdate downloads the new version, replaces the exe, and restarts.
func (a *App) ApplyUpdate(downloadURL string) string {
	err := updater.ApplyUpdate(downloadURL, func() {
		wailsrt.Quit(a.ctx)
	})
	if err != nil {
		slog.Error("self-update failed", "error", err)
		return err.Error()
	}
	return ""
}

// --- System tray ---

// onSystrayReady is called by the systray library when the tray icon is ready.
func (a *App) onSystrayReady() {
	systray.SetIcon(tray.Icon())
	systray.SetTooltip("SC Bridge Companion")

	mShow := systray.AddMenuItem("Show", "Show SC Bridge Companion")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit SC Bridge Companion")

	for {
		select {
		case <-mShow.ClickedCh:
			wailsrt.WindowShow(a.ctx)
		case <-mQuit.ClickedCh:
			systray.Quit()
			wailsrt.Quit(a.ctx)
		}
	}
}

// minimizeWatcher polls for window minimization and hides to tray when enabled.
func (a *App) minimizeWatcher(ctx context.Context) {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.mu.Lock()
			minimize := a.cfg != nil && a.cfg.MinimizeToTray
			a.mu.Unlock()
			if minimize && wailsrt.WindowIsMinimised(a.ctx) {
				wailsrt.WindowHide(a.ctx)
			}
		}
	}
}

// beforeClose is called by Wails when the user clicks the window close button.
// If minimize-to-tray is enabled, hides the window instead of quitting.
func (a *App) beforeClose(ctx context.Context) bool {
	a.mu.Lock()
	minimize := a.cfg != nil && a.cfg.MinimizeToTray
	a.mu.Unlock()
	if minimize {
		wailsrt.WindowHide(ctx)
		return true // prevent quit
	}
	return false // allow quit
}

// --- System settings ---

// GetMinimizeToTray returns whether minimize-to-tray is enabled.
func (a *App) GetMinimizeToTray() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.cfg != nil && a.cfg.MinimizeToTray
}

// SetMinimizeToTray enables or disables minimize-to-tray and persists the setting.
func (a *App) SetMinimizeToTray(enabled bool) {
	a.mu.Lock()
	a.cfg.MinimizeToTray = enabled
	a.cfg.Save(a.cfgPath)
	a.mu.Unlock()
}

// GetStartWithWindows reports whether the app is registered to launch at Windows startup.
func (a *App) GetStartWithWindows() bool {
	out, err := exec.Command("reg", "query",
		`HKCU\Software\Microsoft\Windows\CurrentVersion\Run`,
		"/v", "SC Bridge Companion",
	).Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "SC Bridge Companion")
}

// SetStartWithWindows adds or removes the Windows startup registry entry.
func (a *App) SetStartWithWindows(enabled bool) error {
	if enabled {
		exePath, err := os.Executable()
		if err != nil {
			return err
		}
		return exec.Command("reg", "add",
			`HKCU\Software\Microsoft\Windows\CurrentVersion\Run`,
			"/v", "SC Bridge Companion",
			"/t", "REG_SZ",
			"/d", `"`+exePath+`"`,
			"/f",
		).Run()
	}
	return exec.Command("reg", "delete",
		`HKCU\Software\Microsoft\Windows\CurrentVersion\Run`,
		"/v", "SC Bridge Companion",
		"/f",
	).Run()
}
