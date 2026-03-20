package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/SC-Bridge/sc-companion/internal/config"
	"github.com/SC-Bridge/sc-companion/internal/events"
	"github.com/SC-Bridge/sc-companion/internal/grpcproxy"
	"github.com/SC-Bridge/sc-companion/internal/logtailer"
	"github.com/SC-Bridge/sc-companion/internal/store"
	"github.com/SC-Bridge/sc-companion/internal/sync"
	"github.com/SC-Bridge/sc-companion/internal/tray"
)

func main() {
	logPath := flag.String("log", "", "path to Game.log (auto-detected if empty)")
	configPath := flag.String("config", "config.yaml", "path to config file")
	replay := flag.Bool("replay", false, "process entire log file from start instead of tailing")
	dbPath := flag.String("db", "", "path to SQLite database (default: ~/.scbridge/companion.db)")
	proxyPort := flag.Int("proxy-port", 0, "gRPC proxy port (default: 8443, 0 = use config)")
	noProxy := flag.Bool("no-proxy", false, "disable gRPC proxy")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Warn("no config file, using defaults", "path", *configPath, "error", err)
		cfg = config.Default()
	}

	if *logPath != "" {
		cfg.LogPath = *logPath
	}

	if cfg.LogPath == "" {
		detected := config.DetectGameLog()
		if detected == "" {
			slog.Error("could not find Game.log — pass --log or set log_path in config")
			os.Exit(1)
		}
		cfg.LogPath = detected
		slog.Info("auto-detected Game.log", "path", cfg.LogPath)
	}

	// Resolve database path
	resolvedDBPath := *dbPath
	if resolvedDBPath == "" {
		home, _ := os.UserHomeDir()
		dir := filepath.Join(home, ".scbridge")
		os.MkdirAll(dir, 0700)
		resolvedDBPath = filepath.Join(dir, "companion.db")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		slog.Info("shutting down...")
		cancel()
	}()

	// Open local database
	db, err := store.New(resolvedDBPath)
	if err != nil {
		slog.Error("failed to open database", "path", resolvedDBPath, "error", err)
		os.Exit(1)
	}
	defer db.Close()
	slog.Info("database opened", "path", resolvedDBPath)

	// Event bus — all parsed events flow through here
	bus := events.NewBus()

	// Deduplication + multi-line coalescing
	dedup := events.NewDeduplicator(10 * time.Second)
	coalesce := events.NewCoalesceMultiLine()

	// Tray controller for status tracking
	trayCtrl := tray.NewController(bus, db, cancel)

	// Store subscriber — persist deduplicated events to SQLite
	bus.Subscribe(func(evt events.Event) {
		// Run through coalescer first (handles multi-line money transfers)
		merged, emit := coalesce.Process(evt)
		if !emit {
			return
		}

		// Skip internal/pending events
		if merged.Type == "money_amount" {
			return
		}

		// Deduplicate
		if dedup.IsDuplicate(merged) {
			return
		}

		// Persist + log
		if _, err := db.InsertEvent(merged); err != nil {
			slog.Error("failed to store event", "type", merged.Type, "error", err)
			return
		}
		slog.Info("event",
			"type", merged.Type,
			"data", fmt.Sprintf("%v", merged.Data),
		)
	})

	// Start API sync client (runs in background goroutine)
	if cfg.APIToken != "" {
		syncClient := sync.NewClient(cfg.APIEndpoint, cfg.APIToken, db)
		go func() {
			if err := syncClient.Run(ctx); err != nil {
				slog.Error("sync client stopped", "error", err)
			}
		}()
		slog.Info("API sync enabled", "endpoint", cfg.APIEndpoint)
	} else {
		slog.Info("API sync disabled — set api_token in config to enable")
	}

	// Start gRPC proxy (runs in background goroutine)
	if *noProxy {
		cfg.ProxyEnabled = false
	}
	if *proxyPort > 0 {
		cfg.ProxyPort = *proxyPort
	}
	if cfg.ProxyPort == 0 {
		cfg.ProxyPort = 8443
	}
	if cfg.ProxyEnabled {
		proxy, err := grpcproxy.NewProxy(grpcproxy.ProxyConfig{
			ListenAddr: fmt.Sprintf("127.0.0.1:%d", cfg.ProxyPort),
			CADir:      config.DataDir(),
		}, bus)
		if err != nil {
			slog.Error("failed to create gRPC proxy", "error", err)
		} else {
			go func() {
				if err := proxy.Run(ctx); err != nil {
					slog.Error("gRPC proxy stopped", "error", err)
				}
			}()
		}
	} else {
		slog.Info("gRPC proxy disabled")
	}

	// Start log tailer
	tailer, err := logtailer.New(cfg.LogPath, bus)
	if err != nil {
		slog.Error("failed to create log tailer", "error", err)
		os.Exit(1)
	}

	slog.Info("SC Bridge Companion starting",
		"log", cfg.LogPath,
		"replay", *replay,
		"db", resolvedDBPath,
	)

	var runErr error
	if *replay {
		runErr = tailer.RunFromStart(ctx)
	} else {
		runErr = tailer.Run(ctx)
	}

	// Print final status
	status := trayCtrl.StatusLine()
	slog.Info("final status", "status", status)

	total, _ := db.TotalEvents()
	counts, _ := db.EventCounts()
	slog.Info("event summary", "total", total)
	for t, c := range counts {
		slog.Info("  event type", "type", t, "count", c)
	}

	if err := runErr; err != nil && ctx.Err() == nil {
		slog.Error("tailer stopped with error", "error", err)
		os.Exit(1)
	}

	slog.Info("SC Bridge Companion stopped")
}
