# Session Journal

A living journal that persists across compactions. Captures decisions, progress, and context.

## Current State
- **Focus:** Session complete. v0.3.14-beta.2 shipped. All changes on `release/v0.3.12` branch, pending PR merge to main.
- **Blocked:** main is branch-protected — PR needed to merge release/v0.3.12 → main. `gh` CLI not available; must be done via GitHub web UI.

## Log

### 2026-03-29 20:30 — Completed: installer fixes, About tab, pre-release pipeline
- **SC-Log-Samples corpus analysis** — catalogued everything SC writes to Game.log at startup: hardware, local IPs, input/audio/VR devices, overlay software, character identity, build info. Wrote `docs/log-data-reference.md`.
- **About tab** (`frontend/src/components/About.jsx`) — two accordion sections: "What Lives in Game.log" (8 data categories, read/sync status per field) and "Events Tracked by Companion" (70+ events, SYNC/LOCAL badges). Privacy note leads with character vs personal identity distinction.
- **MSI version fix** — WiX `!(bind.FileVersion.MainExecutable)` was unreliable; replaced with `-d ProductVersion=` passed from git tag in CI. Windows now shows correct version in Installed Apps.
- **Settings console flash fix** — `GetStartWithWindows`/`SetStartWithWindows` were spawning `reg.exe` causing a console window flash on every Settings open. Replaced with `golang.org/x/sys/windows/registry` direct API calls.
- **Installed Apps icon fix** — `ARPPRODUCTICON` was not set in `installer.wxs`. Added `Icon` element + property, and copy `build/windows/icon.ico` in CI before WiX build.
- **Pre-release CI support** — CI now auto-detects pre-release from tag suffix (hyphen), strips suffix for MSI version, marks GitHub release as pre-release automatically.
- **Shipped:** `v0.3.13` (full release), `v0.3.14-beta.1`, `v0.3.14-beta.2` (pre-releases)
- **Recurring user issue** — one user getting "cannot access device/path/file" — diagnosed as Windows Defender quarantining the unsigned exe. Provided step-by-step remediation. Long-term fix is code signing (SignPath already stubbed in CI).
- Key files changed: `app.go`, `wails.json`, `installer/installer.wxs`, `.github/workflows/build.yml`, `frontend/src/App.jsx`, `frontend/src/components/About.jsx`, `docs/log-data-reference.md`

### 2026-03-24 18:00 — Completed: parser audit and overhaul session
- Ran full corpus analysis (180 log files, Jan–Mar 2026) using subagent + `cmd/logtest` harness
- Fixed 10 bugs: case mismatches, duplicate patterns, multi-line format failures, missing severity tier
- Added 19 new patterns across 6 categories (mission, zone, crime, party, misc, player state)
- Implemented generalised multi-line state machine (`pendingType`/`pendingData`) — covers money_sent, party_member_joined, party_member_left
- Restored `blueprint_received` after user confirmed it fires in their live logs
- All 57 patterns verified against 180-file corpus; zero patterns with zero hits (3 internal continuation patterns intentionally emit under a different type name)
- Created `docs/parser-patterns.csv` — full input/output reference
- Updated `bus.go` EventCategories to reflect all current event types
- Created DEVLOG.md and TODO.md from scratch
- Key files changed: `internal/logtailer/parser.go`, `internal/events/bus.go`, `cmd/logtest/main.go`, `docs/parser-patterns.csv`
- 11 commits on main branch this session
