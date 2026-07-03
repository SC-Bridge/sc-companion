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

### 2026-07-03 — Accountant data-loss fixes: event_id stamping + skip-not-synced

Two verified data-loss issues in the event → SC Bridge accountant path.

**1. Stable `event_id` on every event.** The server-side accountant bridge
(`companion-bridge.ts`) dedupes ledger entries on `data.event_id`, falling back
to `companion:<type>:<timestamp>` when absent. Two same-type economy events in
the same log-timestamp second (e.g. two fines) shared a fallback key, so the
second was silently dropped from the ledger. Now `events.StampID` assigns a
UUIDv4 to `Data["event_id"]` once, in the bus subscriber (`app.go`) — after
dedup (so the always-unique id can't defeat it) and before both the SQLite
insert and the JSONL write, so both and every resend carry the byte-identical
id. Stamping at the subscriber (not the parser) means multi-line events are
already coalesced into one Event, so the merged event gets exactly one id.
`StampID` is idempotent (won't regenerate an existing id), so store resends stay
stable.

**2. Disabled sync-preference types no longer permanently lost.** Previously the
sync client marked *all* fetched events `synced=1`, including preference-filtered
ones — so disabling a type (or having it default-off) permanently withheld those
events from the accountant, even after re-enabling. The `synced` column is now
tri-state (`0` pending, `1` done, `2` skipped; no migration — column was already
INTEGER). `syncBatch` marks disabled-type events `2` and only delivered events
`1`. Re-enabling a type (`SetSyncPreference`) calls `RequeueSkipped` (`2→0`);
`ResetSyncPreferences` calls `RequeueAllSkipped`. The five accountant-critical
economy types were already default-on; a comment now guards them.

Files: `internal/events/id.go` (new), `internal/store/store.go`,
`internal/sync/client.go`, `app.go`, `internal/config/preferences.go`,
`frontend/src/components/Settings.jsx` (Economy toggle consequence copy),
`endpoints.md`. Tests added (first `*_test.go` in the repo): `events/id_test.go`,
`store/store_test.go`, `sync/client_test.go`, `logtailer/parser_test.go`.

### 2026-05-08 — Pipeline audit, dead-code cleanup, endpoint probe

#### Pipeline audit — upload coverage

Cross-referenced every parser-emitted event type against `EventCategories()` (the
Settings UI surface) and confirmed all **57** user-facing types are exposed and
wired through to the sync client. The 6 internal scaffolding types
(`money_sent_pending`, `money_amount`, `party_member_joined_pending`,
`party_join_continuation`, `party_member_left_pending`, `party_left_continuation`)
correctly never reach the bus — `Parser.Parse` returns `(_, false)` for them and
emits the merged form on the continuation line.

Default-on coverage: only **18 of 57** types are enabled in
`DefaultSyncPreferences`. The other 39 are uploadable but require a manual toggle
in Settings. `IsEnabled()` returns `false` for any key not in the map
(`preferences.go:79`), so unknown types never sync; toggling writes the key and
makes them upload from that point forward (no retroactive sync — filtered events
are still marked `synced=1` in `client.go:131-138`).

Conclusion: pipeline is intact, ability to upload is complete for every parsed
type, but defaults are conservative.

#### Cleanup performed

- **Deleted dead code** — `SyncWorthyTypes` map and `Event.IsSyncWorthy()` in
  `internal/events/bus.go` had no callers. Removed.
- **Deleted redundant coalescer** — `CoalesceMultiLine` in
  `internal/events/dedup.go` re-implemented multi-line money merging that
  `Parser.Parse` already handles. Removed the struct + the
  `coalesce.NewCoalesceMultiLine()` / `coalesce.Process(evt)` plumbing in
  `app.go`, including the defensive `if merged.Type == "money_amount" { return }`
  guard that only existed to compensate for the redundant path.
- **Doc fix** — `endpoints.md` "Event types synced by default" table:
  added `insurance_claim_complete` (was missing); removed `blueprint_received`
  (claimed to be default but wasn't); added `Combat: fatal_collision` row.
- **TODO** — removed obsolete "Review `SyncWorthyTypes`" item.

Verified with `go build ./...` and `go vet ./...` — both clean.

#### Endpoint probe (read-only)

Probed `https://scbridge.app` and `https://staging.scbridge.app` against the
endpoints documented in `endpoints.md`. Used the local
`%APPDATA%\SCBridge\auth.json` session token (created 2026-03-27).

| Endpoint | Result |
|---|---|
| `GET /` (prod & staging hosts) | 200 — alive |
| `GET /api/health` | 200 `{"status":"ok"}` |
| `GET /api/status` | 200 — public, returns sync job history (see below) |
| `GET /api` | 404 — no root handler |
| `GET /api/companion/friends` (Bearer) | **401** — token rejected |
| `GET /api/companion/friends` (X-API-Key) | 401 — legacy auth also rejected |
| `POST /api/companion/events` (Bearer, `{"events":[]}`) | 401 — token rejected |
| `POST /api/companion/heartbeat` (Bearer, `{}`) | 401 — token rejected |
| `GET /api/companion/connect?port=…&state=…` | 200 — public OAuth initiator |
| `GET https://api.github.com/repos/SC-Bridge/sc-companion/releases/latest` | 200 — latest is `v0.3.14` (2026-03-29) |

Findings:

1. **Local session token is dead** — verbose curl confirmed the
   `Authorization: Bearer …` header was sent verbatim; server returned
   `{"error":"Authentication required"}`. Token has expired (or was revoked) since
   2026-03-27. Reconnecting via Settings → "Connect to SC Bridge" gets a fresh
   one. The `SetOnAuthExpired` callback (`app.go:258`) is wired to detect this
   and emit `auth_expired` to the frontend.

2. **`/companion/friends` IS deployed** — `endpoints.md:94, 104` claims it
   "returns 404 — endpoint not yet deployed", but the live server returns 401
   (auth gate active), confirming the route exists. Doc is stale — update when
   convenient.

3. **Server-side D1 errors visible in `/api/status`** — `production_status`
   sync job has been failing daily from 2026-05-04 → 2026-05-08 with
   `D1_ERROR: too many SQL variables at offset 251: SQLITE_ERROR`. The
   companion `ships` sync (49 records) runs successfully alongside it. This is
   a backend bug on `scbridge.app` (Cloudflare D1), not a companion issue —
   batched insert/update needs chunking. Worth flagging to the server team.

---

### Unreleased — SC Bridge Suite bundle (2026-04-19)

Added a WiX Burn bootstrapper (`installer/bundle.wxs`) that lets users pick which
SC Bridge tools to install (SC-Companion + SC-HUD, both checked by default). The bundle
ships as `SCBridgeSuite-setup.exe` alongside the existing `SCBridgeCompanion-setup.msi` on
sc-companion releases.

**Architecture:**
- Each child app keeps its own independent MSI, repo, and release pipeline. The bundle
  just chains them — no code is shared, no upgrade entanglement.
- `MsiPackage` entries use `Compressed="no"` + `DownloadUrl` pointing at version-pinned
  GitHub release URLs. The bundle .exe is a small stub (~5 MB); MSIs download at install time.
- Two bundle Variables (`InstallCompanion`, `InstallHud`) bound to checkboxes in a custom
  WixStandardBootstrapperApplication theme (`bundle-theme.xml` + `bundle-theme.wxl`).
  Also overridable from the command line for silent/scripted installs.
- Burn hash-pins each MSI at bundle-build time, so the bundle is always re-cut whenever
  EITHER child repo publishes a new release. SC-HUD's CI fires a `repository_dispatch`
  event (`hud-released`) at sc-companion to trigger a bundle-only rebuild.
- New `build-bundle` job in the CI: skipped on dispatch path's MSI build, downloads both
  child MSIs from their GitHub releases, builds + signs the bundle, uploads it as
  `SCBridgeSuite-setup.exe` to sc-companion's latest release (`--clobber`).

**Setup still needed before this works end-to-end:**
- Add repo secret `SUITE_DISPATCH_TOKEN` in `SC-Bridge/SC-HUD` — PAT (or GitHub App token)
  with `repo` scope on `SC-Bridge/sc-companion`. Without it, SC-HUD releases won't trigger
  bundle rebuilds (manual workflow_dispatch still works as a fallback).
- Configure SignPath project policy `release-signing-bundle` for Burn bundle signing
  (handles engine extract / sign / reattach automatically). Without it, the bundle ships
  unsigned and triggers SmartScreen.

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
