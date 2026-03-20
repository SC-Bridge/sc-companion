package main

import (
	"embed"
	"log/slog"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wailslinux "github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	app := NewApp()

	err := wails.Run(&options.App{
		Title:     "SC Bridge Companion",
		Width:     1400,
		Height:    950,
		MinWidth:  900,
		MinHeight: 600,
		WindowStartState: options.Maximised,
		BackgroundColour: &options.RGBA{
			R: 9, G: 19, B: 31, A: 255, // #09131f — sc-darker
		},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
		Bind: []interface{}{
			app,
		},
		Frameless:            false,
		DisableResize:        false,
		StartHidden:          false,
		HideWindowOnClose:    false,
		Windows: &windows.Options{
			WebviewIsTransparent:              false,
			WindowIsTranslucent:               false,
			DisableWindowIcon:                 false,
			DisableFramelessWindowDecorations: false,
			Theme:                             windows.Dark,
		},
		Linux: &wailslinux.Options{
			ProgramName: "SC Bridge Companion",
		},
	})

	if err != nil {
		slog.Error("wails app failed", "error", err)
		os.Exit(1)
	}
}
