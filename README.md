# SC Bridge Companion

[![Download Portable EXE](https://img.shields.io/badge/Download-Portable%20EXE-2ea44f?style=for-the-badge&logo=windows)](https://github.com/SC-Bridge/sc-companion/releases/latest/download/SCBridgeCompanion-portable.exe)
[![Download MSI Installer](https://img.shields.io/badge/Download-MSI%20Installer-0078d4?style=for-the-badge&logo=windows)](https://github.com/SC-Bridge/sc-companion/releases/latest/download/SCBridgeCompanion-setup.msi)

A Windows desktop companion app for [SC Bridge](https://scbridge.app) that monitors your Star Citizen `Game.log` in real-time, parses game events, and syncs them to your SC Bridge profile.

## Features

- **Real-Time Game Log Monitoring** — Tails `Game.log` with automatic detection (checks launcher log, running processes, registry, and drive scan)
- **57 Event Patterns** — Parses player logins, location changes, ship boarding/exiting, insurance claims, contracts, money transfers, party events, quantum travel, injuries, rewards, crimes, and more
- **Multi-Line Event Coalescing** — Handles compound events that span multiple log lines (money transfers, party joins/leaves)
- **Live Dashboard** — Dark-themed React UI with real-time event feed, friends list, and status display
- **SC Bridge Sync** — OAuth 2.0 authentication syncs events to your SC Bridge profile automatically
- **Friends List** — View your Spectrum friends and online status
- **System Tray** — Runs quietly in the background; minimize to tray with restore on click
- **Start with Windows** — Optional registry-based startup so it's always running when you play
- **Self-Updating** — Checks for updates every 4 hours and installs with one click
- **JSONL Event Log** — Writes events to `events.log` for WingmanAI integration
- **Environment Switching** — Toggle between production and staging with Ctrl+Shift+D

## Installation

### Portable EXE

Download `SCBridgeCompanion-portable.exe` and run it from anywhere. No installation required. Config and data are stored in `%APPDATA%\SC Companion\`.

### MSI Installer

Download `SCBridgeCompanion-setup.msi` for a standard Windows installation to Program Files with Start Menu shortcuts and Add/Remove Programs integration.

### Windows SmartScreen Warning

SC Bridge Companion is **not yet code-signed** (code signing is underway). Windows may show a SmartScreen warning when you first run it:

> "Windows protected your PC" / "Windows cannot access the specified device, path or file"

To fix this:
1. Right-click the downloaded file
2. Click **Properties**
3. Check **Unblock** at the bottom of the General tab
4. Click **OK**

This warning will go away once code signing is in place.

## Requirements

- Windows 10 or later
- Star Citizen installed (the app needs read access to `Game.log`)
- Internet connection (for SC Bridge sync and updates)

## How It Works

```
Game.log
  └─ Log Tailer (real-time tail with session resume)
       └─ Parser (57 regex patterns + multi-line state machine)
            └─ Event Bus (pub/sub with 10s deduplication)
                 ├─ SQLite (local event persistence)
                 ├─ SC Bridge API (OAuth sync)
                 ├─ JSONL Log (WingmanAI integration)
                 └─ React Dashboard (live event feed)
```

The app automatically finds your `Game.log` on startup. If auto-detection fails, you can browse to it manually in Settings. On launch, it seeks to the last `player_login` event to restore your session context.

## Connecting to SC Bridge

1. Open the app and go to the **Settings** tab
2. Click **Connect to SC Bridge**
3. Your browser opens to authorize the app via OAuth
4. Once authorized, events sync automatically to your [scbridge.app](https://scbridge.app) profile

## Tech Stack

- **Backend**: Go 1.25, [Wails v2](https://wails.io)
- **Frontend**: React, Vite, dark theme with cyan accents
- **Storage**: SQLite (modernc.org/sqlite — pure Go)
- **System Tray**: fyne.io/systray
- **Installer**: WiX Toolset 4.x
- **CI/CD**: GitHub Actions (build on tag push)

## Development

### Prerequisites

- Go 1.25+
- Node.js 20+
- [Wails CLI](https://wails.io/docs/gettingstarted/installation)

### Build

```bash
# Install frontend dependencies
cd frontend && npm ci && cd ..

# Development mode (hot reload)
wails dev

# Production build
wails build -platform windows/amd64
```

## License

MIT
