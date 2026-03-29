# What SC Companion Reads From Your Logs

> A reference for the "About" panel and privacy documentation.
> All example values below are anonymized — real values are never stored beyond your local machine unless you opt in to SC Bridge sync.

---

## Player Identity

Logged once per session, near startup.

| Field | Source | Example |
|---|---|---|
| Handle (in-game name) | `<Expect Incoming Connection>` | `YourHandle` |
| Player GEID | `<Expect Incoming Connection>` | `201990621533` |
| Session UUID | `[Notice] AccountLogin` | `3c4fd6dd-bbe3-b943-bb56-8fcfe9f04965` |
| Windows username | `[GameProfiles]` | `(not synced)` |

The handle and GEID are the only identity fields synced to SC Bridge if connected. Session UUIDs are used locally for event correlation only.

---

## Hardware Detected at Launch

Star Citizen logs a full hardware census at startup. The companion reads this for the session header but does **not** sync hardware data.

### CPU

```
AMD Ryzen 7 7700X 8-Core Processor    16 logical cores
13th Gen Intel Core i9-13900K         32 logical cores
AMD Ryzen 7 9800X3D 8-Core Processor  16 logical cores
AMD Ryzen 9 5950X 16-Core Processor   32 logical cores
```

### GPU

```
NVIDIA GeForce RTX 4070 Ti   11994 MB VRAM
NVIDIA GeForce RTX 3090      24325 MB VRAM
NVIDIA GeForce RTX 5090      32187 MB VRAM
NVIDIA GeForce RTX 5080      15977 MB VRAM
```

Secondary adapters (AMD integrated, Microsoft Basic Render Driver) are also logged but ignored.

### RAM & Display

```
Physical RAM:    32 GB – 64 GB (varies by system)
Resolutions:     5120×1440 @ 240Hz  |  5120×2160  |  2560×1440  |  1920×1080
HDR:             Detected where supported (brightness in nits logged)
```

### Performance Index

Logged at startup as CPU/GPU benchmark scores. Used internally by CIG — not parsed.

---

## Network Information

Logged during session initialization.

| Field | Example |
|---|---|
| gRPC server endpoint | `pub-sc-alpha-460-11135423.test1.cloudimperiumgames.com:443` |
| Environment tag | `PUB` or `PTU` |
| UDP game port | `0.0.0.0:64090` |
| Local IPv4 | `192.168.x.x` *(not synced)* |
| Local IPv6 link-local | `fe80::xxxx` *(not synced)* |

Local IP and network interface data are logged by the game. The companion does not read or transmit these fields.

---

## Game Build & Version

| Field | Example |
|---|---|
| Build number | `11010425`, `11135423`, `11494258` |
| Branch | `sc-alpha-4.5.0`, `sc-alpha-4.6.0`, `sc-alpha-4.7.0` |
| Full version string | `4.6.173.39432` |
| Config | `shipping` |
| Environment | `PUB` (Live) / `PTU` (Test) |

Build number appears in the log filename and in the first lines of every session.

---

## Input Devices

Enumerated at launch by DirectInput.

**Example flight control setup:**
```
Joystick 0:  VKBsim Gladiator EVO R     GUID: {0200231D-...}
Joystick 1:  VKBsim Gladiator EVO OT L  GUID: {3201231D-...}
```

Standard mouse/keyboard are also logged. GUIDs are hardware identifiers — not synced.

---

## Audio Devices

Wwise enumerates all audio endpoints at startup.

**Example output devices:**
```
SteelSeries Sonar - Stream
Stereo Mix (Realtek)
```

**Example input (microphone) devices:**
```
Microphone (Arctis Nova Pro Wireless)
Microphone (Realtek)
SteelSeries Sonar - Microphone
```

Not parsed by the companion beyond presence detection.

---

## VR / OpenXR

Logged if a headset runtime is present.

| Runtime | Device |
|---|---|
| `VirtualDesktopXR 1.0.9` | Meta Quest (via Virtual Desktop) |
| `Oculus` | Meta Quest (native runtime) |
| *(none)* | No headset detected |

---

## Overlay & Launcher Software

Detected via Vulkan layer enumeration. Present on various test systems:

```
Steam overlay
OBS (via OBS-Vulkan hook)
Overwolf
ReShade
Epic Online Services (EOS)
GOG Galaxy
Rockstar Social Club
```

The companion does not act on or sync this list.

---

## Installation Path

Logged as the executable path. Used by the companion for Game.log auto-detection only.

```
C:\Roberts Space Industries\StarCitizen\LIVE\Bin64\StarCitizen.exe
C:\Program Files\Roberts Space Industries\StarCitizen\LIVE\Bin64\StarCitizen.exe
D:\Star Citizen\StarCitizen\LIVE\Bin64\StarCitizen.exe
```

PTU installs use a `\PTU\` path instead of `\LIVE\`.

---

## In-Game Events Parsed (Active)

These are the events the companion actively extracts and (optionally) syncs. See `parser-patterns.csv` for full regex detail.

| Category | Events |
|---|---|
| **Player** | Login, handle/GEID capture |
| **Ships** | Board, exit, ship name + owner |
| **Contracts** | Accepted, completed, failed |
| **Quantum travel** | Target selected, arrived |
| **Location** | Zone entered, inventory location |
| **Jurisdiction** | UEE jurisdiction entered |
| **Armistice** | Entered, exiting, exited |
| **Monitored space** | Entered, exited, down, restored |
| **Crime** | CrimeStat change, crime committed, fined, vehicle impounded |
| **Economy** | Money sent, transaction complete, insurance claim/complete, refinery complete |
| **Player status** | Injury (severity/body part/tier), incapacitated |
| **Items** | Blueprint received |

---

## What Is NOT Collected

- Local IP addresses or network interface data
- Hardware identifiers (GPU device ID, joystick GUIDs, audio device IDs)
- Windows username or hostname
- Overlay/launcher software list
- Raw log lines
- Any data from sessions where SC Bridge sync is disabled

---

## Sample Build Timeline (Jan–Mar 2026)

Illustrates the build cadence observed across the log corpus.

```
11010425  sc-alpha-4.5.0   09 Jan 2026
11135423  sc-alpha-4.6.0   28 Jan 2026
11218823  sc-alpha-4.6.x   09 Feb 2026
11303722  sc-alpha-4.6.x   24 Feb 2026
11319298  sc-alpha-4.6.x   26 Feb 2026
11377160  sc-alpha-4.6.x   05 Mar 2026
11429312  sc-alpha-4.6.x   12 Mar 2026
11450623  sc-alpha-4.6.x   14 Mar 2026
11494258  sc-alpha-4.7.0   21 Mar 2026  ← PTU
```

Build number appears in both the filename and the log header, making it reliable for version-gating event parsers.
