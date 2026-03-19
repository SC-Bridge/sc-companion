package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/SC-Bridge/sc-companion/internal/config"
	"github.com/SC-Bridge/sc-companion/internal/events"
	"github.com/SC-Bridge/sc-companion/internal/logtailer"
)

func main() {
	logPath := flag.String("log", "", "path to Game.log (auto-detected if empty)")
	configPath := flag.String("config", "config.yaml", "path to config file")
	replay := flag.Bool("replay", false, "process entire log file from start instead of tailing")
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

	// Event bus — all parsed events flow through here
	bus := events.NewBus()

	// Start log tailer
	tailer, err := logtailer.New(cfg.LogPath, bus)
	if err != nil {
		slog.Error("failed to create log tailer", "error", err)
		os.Exit(1)
	}

	// Log subscriber for debugging
	bus.Subscribe(func(evt events.Event) {
		slog.Info("event",
			"type", evt.Type,
			"source", evt.Source,
			"data", fmt.Sprintf("%v", evt.Data),
		)
	})

	slog.Info("SC Bridge Companion starting", "log", cfg.LogPath, "replay", *replay)

	var runErr error
	if *replay {
		runErr = tailer.RunFromStart(ctx)
	} else {
		runErr = tailer.Run(ctx)
	}
	if err := runErr; err != nil && ctx.Err() == nil {
		slog.Error("tailer stopped with error", "error", err)
		os.Exit(1)
	}

	slog.Info("SC Bridge Companion stopped")
}
