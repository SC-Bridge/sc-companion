# SC Companion — Development Log

SC Companion is a Windows desktop application (Wails + Go + React) that tails the Star Citizen `Game.log` file, parses structured events, and syncs them to the SC Bridge API. It lives in the system tray and restores session state on launch.

---

## Architecture

```
Game.log
  └─ logtailer.Tailer        — tail + seekToLastSession on startup
       └─ logtailer.Parser   — regex patterns, multi-line state machine
            └─ events.Bus    — pub/sub, all events flow through here
                 ├─ store.Store        — SQLite dedup + persistence
                 ├─ sync.Client        — HTTP sync to SC Bridge API
                 └─ Wails frontend     — React dashboard, live event feed
```

**Key packages:**
| Package | Purpose |
|---|---|
| `internal/logtailer` | Log tail, parser, multi-line state machine |
| `internal/events` | Event types, bus, sync-worthy registry, category definitions |
| `internal/store` | SQLite persistence and deduplication |
| `internal/sync` | SC Bridge API client |
| `internal/config` | YAML config, Game.log auto-detection (4 strategies) |
| `internal/auth` | OAuth flow |
| `internal/tray` | System tray controller |
| `internal/updater` | Self-update via GitHub releases |
| `cmd/logtest` | Dev tool — runs parser against a log directory |

**Stack:** Go 1.25, Wails v2, React, SQLite (modernc), fyne systray

---

## Changelog

### Unreleased (2026-03-29)
- Fixed wrong icon in Windows Settings → Installed Apps — `ARPPRODUCTICON` was not set in `installer.wxs`, causing Windows to fall back to a generic icon. Added `Icon` element + `ARPPRODUCTICON` property and copy `build/windows/icon.ico` into the installer directory in CI before the WiX build step.
- Fixed console window flashing briefly every time Settings tab was opened — `GetStartWithWindows` / `SetStartWithWindows` were spawning `reg.exe` as a child process. Replaced with direct `golang.org/x/sys/windows/registry` API calls.
- Clarified character identity vs personal identity in About tab — renamed "Player Identity" → "Character Identity", updated field labels and privacy note to make explicit that handle/GEID identify the in-game avatar, not the person.
- Fixed MSI installer registering wrong version in Windows Apps list (was showing 0.3.9.0 even when installing v0.3.12 MSI from GitHub release).
  - Root cause: WiX `Version="!(bind.FileVersion.MainExecutable)"` binds to the PE FileVersion resource, which Wails does not reliably update from `wails.json productVersion` when built with `-ldflags`.
  - Fix: changed `installer.wxs` to use `Version="$(ProductVersion)"` (a WiX define) and added `-d ProductVersion=<tag>` to the CI `wix build` command so the version flows directly from the git tag with no PE resource dependency.

### Unreleased (2026-03-29)
- Analysed SC-Log-Samples corpus (~180 files, 5 named players) to catalogue everything Star Citizen writes to `Game.log` at startup and during play — player identity, full hardware specs, local IPs, input/audio/VR devices, overlay software, build info.
- Added `docs/log-data-reference.md` — anonymised reference of all data categories in the log, what the companion reads, and what gets synced.
- Added **About tab** (`frontend/src/components/About.jsx`):
  - Stats row: total event types tracked, synced count, local-only count
  - "What Lives in Game.log" — accordion with 8 data categories (Identity, Hardware, Network, Build, Input, Audio, VR, Overlays); each field shows read/not-read and sync status
  - "Events Tracked by Companion" — all 8 event categories expandable, each event shows fields captured and SYNC/LOCAL badge
  - Privacy note explicitly listing data classes that never leave the machine

### v0.3.9 (2026-03-27)
- Fixed self-update silently failing for MSI installs in `C:\Program Files` — PowerShell ran without elevation so `Copy-Item` was denied. Now uses `msiexec /passive -Verb RunAs` which triggers a UAC prompt and installs correctly.
- Fixed portable exe update timing race — replaced `Start-Sleep -Seconds 2` with `$p.WaitForExit(30000)` on the actual process PID so the file lock is guaranteed released before copy.
- Frontend now prefers `installerUrl` over `downloadUrl` when both are available, routing MSI installs to the correct update path.

### v0.3.8 (2026-03-26)
- Investigated OAuth connection flow — root cause identified as a server-side change to `/companion/connect` page (JS fetch replacing traditional form submit, breaking the 302 redirect-to-localhost callback). App code is correct; fix required on scbridge.app website.
- Identified bug: `ConnectToSCBridge` (app.go:604) assigns `svcCancel` to `a.cancel` instead of `a.syncCancel`, overwriting the app-level service context cancel. `startSync` corrects `a.syncCancel` itself so sync works, but shutdown cleanup is affected. Needs fix.

### v0.3.4 (2026-01-XX)
- System tray support with minimize-to-tray
- Startup on Windows login
- Smarter Game.log detection (4 strategies: launcher log, running process, registry, drive scan)
- Browse button for manual log path override
- Dark tray icon
- Windows icon cache refresh after self-update

### v0.3.3
- SignPath code signing integration
- Browser extension links on Dashboard

### v0.3.2
- Self-update via GitHub releases (PowerShell, hidden window)

### v0.3.1
- Dark icon background, invisible update, browser icons, update banner

### v0.3.0
- gRPC interceptor + Wails desktop app with React UI
- Live event feed dashboard

### v0.2.0
- SQLite store, dedup, gRPC proxy, API sync, tray controller

### v0.1.0
- Initial scaffold — log tailer with 29 event parsers

---

## 2026-03-24 — Parser audit and overhaul

Ran a comprehensive analysis of all parser patterns against 180 log files (Jan–Mar 2026, builds 11010425–11494258, ~180k lines per file). Identified and fixed all bugs, added 19 missing patterns, and built a test harness.

### Bugs fixed

| Bug | Fix |
|---|---|
| `rewards_earned` — case mismatch (`Earned`/`Rewards` vs `earned`/`rewards`) | Corrected to lowercase |
| `injury` — missing `Moderate` severity in alternation | Added `Moderate` to `(Minor\|Moderate\|Major\|Severe)` |
| `qt_destination_selected` — duplicate of `qt_target_selected` | Removed duplicate |
| `qt_arrived_final` — duplicate of `qt_arrived` | Removed duplicate |
| `blueprint_received` — removed incorrectly (zero corpus hits but confirmed real) | Restored |
| `money_sent` — multi-line: amount on next line with timestamp prefix broke `^\s*(\d+)\s+aUEC\s*$` | Fixed regex; implemented `pendingType`/`pendingData` state machine |
| `ship_boarded` / `ship_exited` — zero hits; closing `"` is on line 2, not line 1 | Removed `"` from end of pattern |
| `objective_complete` — zero hits; same multi-line split | Changed to `(?::\s*"\|$)` terminator |
| `crime_committed` — partial (1/13); mixed single/multi-line | Changed to `(?::\s*"\|$)` terminator |
| `party_member_joined` / `party_member_left` — player name on line 2 | Implemented as 2-line state machine; captures `player` field |

### Multi-line state machine

Three notification types span two log lines. The parser buffers state between lines:

| Event | Line 1 trigger | Line 2 pattern | Emits |
|---|---|---|---|
| `money_sent` | `Added notification "You sent NAME:` | `<ts> AMOUNT aUEC` | `money_sent{recipient, amount, currency}` |
| `party_member_joined` | `Added notification "New Member Joined` | `<ts> NAME has joined the (channel\|group\|party)` | `party_member_joined{player}` |
| `party_member_left` | `Added notification "Member Left` | `<ts> NAME has left the party.` | `party_member_left{player}` |

### New patterns added (19)

**Mission/contract:** `contract_shared`, `objective_complete`, `objective_withdrawn`
**Zone transitions:** `armistice_exiting`, `private_property_entered/exited`, `restricted_area_warning/exited`, `monitored_space_down/restored`
**Crime:** `crime_committed`
**Party:** `party_member_joined`, `party_member_left`, `party_disbanded`
**Misc:** `low_fuel`, `journal_entry_added`
**Player state:** `player_spawned`, `actor_death`, `med_bed_heal`

### Pattern reference

Full input/output reference for all 57 patterns: `docs/parser-patterns.csv`

### Final corpus results (180 files)

All 57 patterns fire. Notable counts:
`player_login` 4599 · `location_change` 1260 · `ship_list_fetched` 1403 · `insurance_claim` 752 · `contract_accepted` 570 · `new_objective` 525 · `player_spawned` 657 · `contract_shared` 288 · `objective_complete` 287 · `ship_boarded` 330 · `party_member_joined` 64

---

## API Reference

**SC Bridge API base:** `https://scbridge.app/api`

**Sync-worthy event types** (sent to API):
`player_login`, `server_joined`, `ship_boarded`, `ship_exited`, `insurance_claim`, `insurance_claim_complete`, `contract_accepted`, `contract_completed`, `contract_failed`, `mission_ended`, `location_change`, `jurisdiction_entered`, `money_sent`, `fined`, `transaction_complete`
