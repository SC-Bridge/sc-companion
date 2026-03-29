# SC Companion — TODO

---

## In Progress / Next

- [ ] Review `SyncWorthyTypes` in `bus.go` — new events from today's parser work may be worth syncing (e.g. `crime_committed`, `actor_death`, `objective_complete`)
- [ ] Frontend: surface new event types in the dashboard (19 new types added, UI categories already updated in `EventCategories()`)
- [ ] About tab: keep `SYNC_WORTHY` set in `About.jsx` in sync with `bus.go` `SyncWorthyTypes` — currently duplicated; consider exposing via a `GetSyncWorthyTypes` Wails binding

---

## Parser

### Known gaps (unmatched `Added notification` types from corpus)
| Count | Notification | Notes |
|---|---|---|
| 642 | `Party` (sub-types) | `Party Launch Accepted`, `Party Launch`, `Party Disbanded` partially covered |
| 124 | `Medical Bed` | Tutorial hints, not actionable |
| 86 | `Downloading` | Hacking/mission download objective state |
| 83 | `Quantum Drive - Spooling` | QT state notification |
| 51 | `Joined hangar queue` | Hangar queue position |
| 46 | `Monitored Space` | Tutorial hints |
| 41 | `Medical Device` | Tutorial hints |
| 29 | `Quantum Travel` | Various QT sub-states |
| 28 | `Ship Startup` | Ship startup sequence |
| 24 | `Exit Bed` | Player exited med bed (distinct from `med_bed_heal`) |
| 20 | `Quantum Travel - Calibration` | QT calibration state |
| 19 | `New Party Leader` | Party leadership changed |
| 14 | `Friend Added` | Social event |
| 13 | `Radar's Ping` | Radar ping notification |
| 13 | `Maps` | Map tutorial hint |
| 12 | `Vehicle Retrieval` | Vehicle retrieval request |
| 11 | `Hunger & Thirst` | Survival stat warning |
| 10 | `Stamina` | Survival stat warning |
| 10 | `Chat` | Chat channel events |
| 9 | `Personal Inner Thought (PIT)` | PIT tutorial |
| 9 | `Ship Movement` | Tutorial hint |
| 9 | `Inventory` | Tutorial hint |
| 4 | `Rewards Retrieval Failed` | Backend failure |
| 3 | `Alliance Aid Collection Rewards - Tier N` | Tier reward |

### Verified patterns (all firing correctly as of 2026-03-24)
- All 57 patterns match at least once across 180 log files
- `money_amount`, `party_join_continuation`, `party_left_continuation` show as zero-hit in `cmd/logtest` — this is expected; they are internal continuation patterns that emit under a different event type name

### Multi-line state machine patterns
Three events require 2-line parsing (pendingType/pendingData):
- `money_sent` — line 2 has amount
- `party_member_joined` — line 2 has player name (3 sub-variants: party / channel / group)
- `party_member_left` — line 2 has player name

---

## Known Issues

- `money_sent` corpus count is only 2 — the log corpus has limited player-to-player transfer data. Pattern is correct; confirmed against known log lines.
- `actor_death` count is only 4 — rare event (requires local player death in destroyed vehicle). Pattern correct.
- `blueprint_received` count is 3 — rare drop. Pattern confirmed against live log sample from 2026-03-22.

---

## Infrastructure / App

- [ ] **Bug:** `ConnectToSCBridge` (app.go:604) — `a.cancel = svcCancel` should be `a.syncCancel = svcCancel`. Overwrites the app-level service context cancel set in startup. Sync works (startSync fixes syncCancel itself) but app shutdown doesn't cancel the original service context (tray, etc.).
- [ ] **Website issue (scbridge.app):** `/companion/connect` Connect button changed from HTML form submit to JavaScript fetch — browser no longer follows the 302 redirect to `localhost:PORT/callback`, so the OAuth token never arrives. Fix on the website side: revert to a plain `<form method="POST">` with no JS interception.
- [ ] Investigate whether `DetectedLogPath()` == `DetectGameLog()` — they're identical functions; one may be redundant
- [ ] `config.go` strategy comment says "Strategy 1" twice (running process and registry) — minor documentation inconsistency

---

## Completed

- [x] Fixed wrong icon in Windows Installed Apps — added ARPPRODUCTICON to WiX installer (2026-03-29)
- [x] Fixed console window flash on Settings open — replaced reg.exe exec with registry API (2026-03-29)
- [x] Clarified character vs personal identity in About tab (2026-03-29)
- [x] Fixed MSI version mismatch — WiX now takes version from git tag via `-d ProductVersion=` instead of PE FileVersion binding (2026-03-29)
- [x] SC-Log-Samples corpus analysis — catalogued all data categories in `Game.log` (2026-03-29)
- [x] Added `docs/log-data-reference.md` — anonymised log data reference (2026-03-29)
- [x] Added About tab with log data inventory and event tracking reference (2026-03-29)
- [x] Parser audit against 180 log files (2026-03-24)
- [x] Fix `rewards_earned` case mismatch
- [x] Fix `injury` missing `Moderate` severity
- [x] Remove duplicate `qt_destination_selected` and `qt_arrived_final`
- [x] Implement `money_sent` multi-line state machine
- [x] Fix `ship_boarded` / `ship_exited` multi-line format
- [x] Fix `objective_complete` multi-line format
- [x] Fix `crime_committed` mixed-line format
- [x] Add 19 missing patterns
- [x] Add player name to `party_member_joined` and `party_member_left`
- [x] Restore `blueprint_received` (confirmed real)
- [x] Generalise pending state from single string to `pendingType`/`pendingData`
- [x] Add `cmd/logtest` — corpus analysis tool
- [x] Add `docs/parser-patterns.csv` — input/output reference for all patterns
- [x] Update `EventCategories()` in `bus.go` to reflect all current event types
