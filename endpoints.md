# SC Bridge Companion — API Endpoints

All requests target `https://scbridge.app/api` (production) or `https://staging.scbridge.app/api` (staging), referred to below as `{API}`.

---

## Outbound Endpoints

| Endpoint | Method | Auth | Purpose | Frequency |
|----------|--------|------|---------|-----------|
| `{API}/companion/events` | POST | Bearer | Batch sync game events | Every 10s (when unsynced events exist) |
| `{API}/companion/heartbeat` | POST | Bearer | Status ping | On-demand |
| `{API}/companion/friends` | GET | Bearer | Full friends list | On component mount |
| `{API}/companion/friends?since=…` | GET | Bearer | Friends delta | Every 30s |
| `{base}/companion/connect?port=…&state=…` | GET | None | OAuth initiation (opens in browser) | On user "Connect" click |
| `https://api.github.com/repos/SC-Bridge/sc-companion/releases/latest` | GET | None | Update check | On-demand |

Auth header is `Authorization: Bearer {session_token}`. Legacy API key connections use `X-API-Key: {api_token}` instead.

---

## POST /companion/events

Batches up to 100 unsynced events and posts them every 10 seconds. Only event types enabled in sync preferences are included; excluded types are marked **skipped** (not synced) so re-enabling the type requeues them for delivery. Every event carries a stable `data.event_id` (UUIDv4, stamped once at first persistence) that the server uses for idempotent dedup.

**Request**
```json
{
  "events": [
    {
      "type": "ship_boarded",
      "source": "log",
      "timestamp": "2026-04-09T01:26:04.000000000Z",
      "data": {
        "ship": "MISC_Freelancer",
        "location": "Stanton"
      }
    }
  ]
}
```

**Response** — `200 OK` (body discarded). `401` triggers auth-expired callback and halts sync.

**Event types synced by default:**

| Category | Types |
|----------|-------|
| Session | `player_login`, `server_joined` |
| Ships | `ship_boarded`, `ship_exited`, `insurance_claim`, `insurance_claim_complete` |
| Contracts | `contract_accepted`, `contract_completed`, `contract_failed`, `mission_ended` |
| Location | `location_change`, `jurisdiction_entered` |
| Economy | `money_sent`, `fined`, `transaction_complete`, `rewards_earned`, `refinery_complete` |
| Combat | `fatal_collision` |

---

## POST /companion/heartbeat

On-demand status ping. Body is a freeform state map.

**Request**
```json
{
  "key": "value"
}
```

**Response** — `200 OK` (body discarded).

---

## GET /companion/friends

Returns the full friends list. Called once on component mount.

**Response** — `200 OK`
```json
{
  "ok": true,
  "friends": [
    {
      "account_id": "string",
      "nickname": "string",
      "display_name": "string",
      "presence": "online | away | offline",
      "activity_state": "string",
      "activity_detail": "string",
      "updated_at": "RFC3339"
    }
  ]
}
```

> **Status:** Returns `404` — endpoint not yet deployed on the server.

---

## GET /companion/friends?since={iso_timestamp}

Returns only friends updated after the given timestamp. Polled every 30 seconds after initial load. The client merges results into the existing list by `account_id`.

Same response shape as full friends list.

> **Status:** Returns `404` — endpoint not yet deployed on the server.

---

## OAuth Flow

1. App opens browser to `{base}/companion/connect?port={random_port}&state={random_state}`
2. User authenticates on SC Bridge
3. Server redirects to `http://127.0.0.1:{random_port}/callback?token={session_token}&state={state}`
4. Local HTTP server validates state, captures token
5. Token saved to `{DataDir}/auth.json`

---

## Event Pipeline

```
Game.log
  → LogTailer (file reader + regex parser)
  → Event { type, source, timestamp, data map[string]string }
  → StampID — data.event_id = UUIDv4 (once, before persistence)
  → SQLite (synced = 0 / pending)
  → SyncClient ticker (10s)
  → POST /companion/events (batch ≤ 100)
  → SQLite (synced = 1 / done, on 200 OK)
```

The `synced` column is tri-state: `0` pending, `1` done, `2` skipped. Events whose type is disabled in sync preferences are marked `2` (skipped) rather than `1`, so re-enabling the type (`RequeueSkipped`) returns them to pending and they sync on the next tick. The server dedupes idempotently on `data.event_id` and orders by the log timestamp, so late delivery is safe.
