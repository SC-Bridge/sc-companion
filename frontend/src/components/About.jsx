import { useState } from 'react'
import { ChevronDown, ChevronRight, Shield, Eye, EyeOff, Cpu, Monitor, Wifi, Gamepad2, Volume2, Headset, Layers, Tag } from 'lucide-react'

// Sync-worthy types must stay in sync with internal/events/bus.go SyncWorthyTypes
const SYNC_WORTHY = new Set([
  'player_login', 'server_joined',
  'ship_boarded', 'ship_exited', 'insurance_claim', 'insurance_claim_complete',
  'contract_accepted', 'contract_completed', 'contract_failed', 'mission_ended',
  'location_change', 'jurisdiction_entered',
  'money_sent', 'fined', 'transaction_complete', 'rewards_earned', 'refinery_complete', 'blueprint_received',
])

// All event categories — mirrors EventCategories() in internal/events/bus.go
const EVENT_CATEGORIES = [
  {
    name: 'Session',
    events: [
      { type: 'player_login', label: 'Player Login', fields: 'handle, geid' },
      { type: 'server_joined', label: 'Server Joined', fields: 'shard' },
      { type: 'player_spawned', label: 'Player Spawned', fields: '' },
      { type: 'entitlement_reconciliation', label: 'Entitlement Reconciliation', fields: 'status, phase' },
    ],
  },
  {
    name: 'Ships',
    events: [
      { type: 'ship_boarded', label: 'Ship Boarded', fields: 'ship, owner' },
      { type: 'ship_exited', label: 'Ship Exited', fields: 'ship, owner' },
      { type: 'insurance_claim', label: 'Insurance Claim', fields: 'request_id' },
      { type: 'insurance_claim_complete', label: 'Claim Complete', fields: 'result' },
      { type: 'vehicle_impounded', label: 'Vehicle Impounded', fields: 'reason' },
      { type: 'hangar_ready', label: 'Hangar Ready', fields: '' },
      { type: 'ship_list_fetched', label: 'Ship List Fetched', fields: 'count' },
      { type: 'ships_loaded', label: 'Ships Loaded', fields: 'count' },
      { type: 'fatal_collision', label: 'Fatal Collision', fields: 'vehicle, zone' },
      { type: 'low_fuel', label: 'Low Fuel', fields: '' },
    ],
  },
  {
    name: 'Missions',
    events: [
      { type: 'contract_accepted', label: 'Contract Accepted', fields: 'name' },
      { type: 'contract_completed', label: 'Contract Completed', fields: 'name' },
      { type: 'contract_failed', label: 'Contract Failed', fields: 'name' },
      { type: 'contract_available', label: 'Contract Available', fields: 'name' },
      { type: 'contract_shared', label: 'Contract Shared', fields: 'name' },
      { type: 'mission_ended', label: 'Mission Ended', fields: 'mission_id, state' },
      { type: 'end_mission', label: 'End Mission', fields: 'mission_id, completion_type, reason' },
      { type: 'new_objective', label: 'New Objective', fields: 'name' },
      { type: 'objective_complete', label: 'Objective Complete', fields: 'description' },
      { type: 'objective_withdrawn', label: 'Objective Withdrawn', fields: 'description' },
    ],
  },
  {
    name: 'Location',
    events: [
      { type: 'location_change', label: 'Location Change', fields: 'player, location' },
      { type: 'jurisdiction_entered', label: 'Jurisdiction Entered', fields: 'jurisdiction' },
      { type: 'armistice_entered', label: 'Armistice Entered', fields: '' },
      { type: 'armistice_exiting', label: 'Armistice Exiting', fields: '' },
      { type: 'armistice_exited', label: 'Armistice Exited', fields: '' },
      { type: 'monitored_space_entered', label: 'Monitored Space Entered', fields: '' },
      { type: 'monitored_space_exited', label: 'Monitored Space Exited', fields: '' },
      { type: 'monitored_space_down', label: 'Monitored Space Down', fields: '' },
      { type: 'monitored_space_restored', label: 'Monitored Space Restored', fields: '' },
      { type: 'private_property_entered', label: 'Private Property Entered', fields: '' },
      { type: 'private_property_exited', label: 'Private Property Exited', fields: '' },
      { type: 'restricted_area_warning', label: 'Restricted Area Warning', fields: '' },
      { type: 'restricted_area_exited', label: 'Restricted Area Exited', fields: '' },
    ],
  },
  {
    name: 'Quantum Travel',
    events: [
      { type: 'qt_target_selected', label: 'Target Selected', fields: 'destination' },
      { type: 'qt_fuel_requested', label: 'Fuel Requested', fields: 'destination' },
      { type: 'qt_arrived', label: 'Arrived', fields: '' },
    ],
  },
  {
    name: 'Economy',
    events: [
      { type: 'money_sent', label: 'Money Sent', fields: 'recipient, amount, currency' },
      { type: 'fined', label: 'Fined', fields: 'amount, currency' },
      { type: 'transaction_complete', label: 'Transaction Complete', fields: '' },
      { type: 'rewards_earned', label: 'Rewards Earned', fields: 'count' },
      { type: 'refinery_complete', label: 'Refinery Complete', fields: 'location' },
      { type: 'blueprint_received', label: 'Blueprint Received', fields: 'name' },
    ],
  },
  {
    name: 'Combat & Health',
    events: [
      { type: 'injury', label: 'Injury', fields: 'severity, body_part, tier' },
      { type: 'incapacitated', label: 'Incapacitated', fields: '' },
      { type: 'actor_death', label: 'Actor Death', fields: 'actor, zone' },
      { type: 'med_bed_heal', label: 'Med Bed Heal', fields: 'bed_name, vehicle, limbs' },
      { type: 'crimestat_increased', label: 'CrimeStat Increased', fields: '' },
      { type: 'crime_committed', label: 'Crime Committed', fields: 'crime' },
      { type: 'emergency_services', label: 'Emergency Services', fields: '' },
      { type: 'journal_entry_added', label: 'Journal Entry Added', fields: 'entry' },
    ],
  },
  {
    name: 'Party',
    events: [
      { type: 'party_member_joined', label: 'Member Joined', fields: 'player' },
      { type: 'party_member_left', label: 'Member Left', fields: 'player' },
      { type: 'party_disbanded', label: 'Party Disbanded', fields: '' },
    ],
  },
]

// What Star Citizen writes to disk — grouped by category
// read: true = companion reads this field  |  synced: true = sent to SC Bridge
const LOG_CONTENTS = [
  {
    id: 'identity',
    label: 'Character Identity',
    icon: Shield,
    items: [
      { label: 'Character handle', read: true, synced: true, note: 'Your in-game avatar name' },
      { label: 'Character GEID', read: true, synced: true, note: 'RSI game account ID, not personal data' },
      { label: 'Session UUID', read: true, synced: false, note: 'Local correlation only' },
      { label: 'Windows username', read: false, synced: false, note: 'Logged but not read' },
    ],
  },
  {
    id: 'hardware',
    label: 'System Hardware',
    icon: Cpu,
    items: [
      { label: 'CPU model & core count', read: false, synced: false, note: 'e.g. Ryzen 7 7700X, 16 cores' },
      { label: 'GPU model & VRAM', read: false, synced: false, note: 'e.g. RTX 4070 Ti, 11994 MB' },
      { label: 'Physical RAM', read: false, synced: false, note: 'e.g. 64 GB' },
      { label: 'Display resolution & Hz', read: false, synced: false, note: 'e.g. 5120×1440 @ 240 Hz' },
      { label: 'HDR capability & brightness', read: false, synced: false, note: 'Peak nits if supported' },
      { label: 'CPU & GPU benchmark scores', read: false, synced: false, note: 'Performance index at launch' },
    ],
  },
  {
    id: 'network',
    label: 'Network',
    icon: Wifi,
    items: [
      { label: 'Local IPv4 address', read: false, synced: false, note: 'e.g. 192.168.x.x' },
      { label: 'Local IPv6 link-local', read: false, synced: false, note: 'fe80::... interface' },
      { label: 'CIG gRPC server endpoint', read: true, synced: false, note: 'pub-sc-alpha-*.cloudimperiumgames.com' },
      { label: 'UDP game port', read: false, synced: false, note: '0.0.0.0:64090' },
    ],
  },
  {
    id: 'display',
    label: 'Game Build & Environment',
    icon: Monitor,
    items: [
      { label: 'Build number', read: true, synced: false, note: 'e.g. 11377160' },
      { label: 'Branch & version string', read: true, synced: false, note: 'e.g. sc-alpha-4.6.0 / 4.6.173.39432' },
      { label: 'Environment (PUB / PTU)', read: true, synced: false, note: 'Live or test server' },
      { label: 'Installation path', read: false, synced: false, note: 'Used for auto-detection only' },
    ],
  },
  {
    id: 'input',
    label: 'Input Devices',
    icon: Gamepad2,
    items: [
      { label: 'Joystick / HOTAS model', read: false, synced: false, note: 'e.g. VKBsim Gladiator EVO' },
      { label: 'Joystick hardware GUIDs', read: false, synced: false, note: 'DirectInput device IDs' },
      { label: 'Mouse button count', read: false, synced: false, note: 'Logged at startup' },
    ],
  },
  {
    id: 'audio',
    label: 'Audio Devices',
    icon: Volume2,
    items: [
      { label: 'Output device names', read: false, synced: false, note: 'All Wwise-enumerated endpoints' },
      { label: 'Microphone device names', read: false, synced: false, note: 'All input endpoints' },
    ],
  },
  {
    id: 'vr',
    label: 'VR / OpenXR',
    icon: Headset,
    items: [
      { label: 'OpenXR runtime name & version', read: false, synced: false, note: 'e.g. VirtualDesktopXR 1.0.9' },
      { label: 'Headset type', read: false, synced: false, note: 'e.g. Meta Quest 3' },
    ],
  },
  {
    id: 'overlays',
    label: 'Overlay & Launcher Software',
    icon: Layers,
    items: [
      { label: 'Vulkan overlay layers', read: false, synced: false, note: 'Steam, OBS, Overwolf, ReShade, EOS, GOG, etc.' },
    ],
  },
]

// ─── Sub-components ──────────────────────────────────────────────────────────

function SyncBadge({ synced }) {
  return (
    <span style={{
      display: 'inline-flex', alignItems: 'center',
      padding: '1px 6px', borderRadius: 4, fontSize: 10,
      fontFamily: 'var(--font-mono)',
      letterSpacing: '0.04em',
      background: synced ? 'rgba(34,211,238,0.1)' : 'rgba(107,114,128,0.15)',
      border: `1px solid ${synced ? 'rgba(34,211,238,0.25)' : 'rgba(107,114,128,0.2)'}`,
      color: synced ? '#22d3ee' : '#6b7280',
    }}>
      {synced ? 'SYNC' : 'LOCAL'}
    </span>
  )
}

function ReadBadge({ read }) {
  if (!read) return (
    <EyeOff size={13} style={{ color: '#374151', flexShrink: 0 }} title="Not read by companion" />
  )
  return (
    <Eye size={13} style={{ color: '#6b7280', flexShrink: 0 }} title="Read by companion" />
  )
}

function LogSection({ section, expanded, onToggle }) {
  const Icon = section.icon
  const readCount = section.items.filter(i => i.read).length
  const total = section.items.length

  return (
    <div style={{
      border: '1px solid rgba(255,255,255,0.06)',
      borderRadius: 10,
      overflow: 'hidden',
    }}>
      <button
        onClick={onToggle}
        style={{
          width: '100%', display: 'flex', alignItems: 'center', gap: 10,
          padding: '11px 14px',
          background: expanded ? 'rgba(255,255,255,0.04)' : 'rgba(255,255,255,0.02)',
          border: 'none', cursor: 'pointer',
          transition: 'background 150ms',
        }}
        onMouseEnter={e => { if (!expanded) e.currentTarget.style.background = 'rgba(255,255,255,0.03)' }}
        onMouseLeave={e => { if (!expanded) e.currentTarget.style.background = 'rgba(255,255,255,0.02)' }}
      >
        <Icon size={14} style={{ color: '#6b7280', flexShrink: 0 }} />
        <span className="font-[family-name:var(--font-display)]" style={{
          flex: 1, textAlign: 'left', fontSize: 12, letterSpacing: '0.05em',
          textTransform: 'uppercase', color: '#d1d5db',
        }}>
          {section.label}
        </span>
        <span style={{ fontSize: 11, color: '#4b5563' }}>
          {readCount}/{total} read
        </span>
        {expanded
          ? <ChevronDown size={13} style={{ color: '#6b7280' }} />
          : <ChevronRight size={13} style={{ color: '#6b7280' }} />
        }
      </button>

      {expanded && (
        <div style={{ borderTop: '1px solid rgba(255,255,255,0.05)' }}>
          {section.items.map((item, i) => (
            <div
              key={i}
              style={{
                display: 'flex', alignItems: 'center', gap: 10,
                padding: '8px 14px',
                borderBottom: i < section.items.length - 1 ? '1px solid rgba(255,255,255,0.04)' : 'none',
                background: 'rgba(0,0,0,0.1)',
              }}
            >
              <ReadBadge read={item.read} />
              <span style={{ flex: 1, fontSize: 12, color: item.read ? '#d1d5db' : '#4b5563' }}>
                {item.label}
              </span>
              {item.synced && <SyncBadge synced />}
              {item.note && (
                <span className="font-[family-name:var(--font-mono)]" style={{ fontSize: 10, color: '#374151' }}>
                  {item.note}
                </span>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

function EventCategoryCard({ category }) {
  const [expanded, setExpanded] = useState(false)
  const syncCount = category.events.filter(e => SYNC_WORTHY.has(e.type)).length

  return (
    <div style={{
      border: '1px solid rgba(255,255,255,0.06)',
      borderRadius: 10,
      overflow: 'hidden',
    }}>
      <button
        onClick={() => setExpanded(p => !p)}
        style={{
          width: '100%', display: 'flex', alignItems: 'center', gap: 10,
          padding: '11px 14px',
          background: expanded ? 'rgba(255,255,255,0.04)' : 'rgba(255,255,255,0.02)',
          border: 'none', cursor: 'pointer',
          transition: 'background 150ms',
        }}
        onMouseEnter={e => { if (!expanded) e.currentTarget.style.background = 'rgba(255,255,255,0.03)' }}
        onMouseLeave={e => { if (!expanded) e.currentTarget.style.background = 'rgba(255,255,255,0.02)' }}
      >
        <span className="font-[family-name:var(--font-display)]" style={{
          flex: 1, textAlign: 'left', fontSize: 12, letterSpacing: '0.05em',
          textTransform: 'uppercase', color: '#d1d5db',
        }}>
          {category.name}
        </span>
        <span style={{ fontSize: 11, color: '#4b5563' }}>
          {category.events.length} events
        </span>
        {syncCount > 0 && (
          <span style={{ fontSize: 10, color: '#22d3ee', opacity: 0.7 }}>
            {syncCount} synced
          </span>
        )}
        {expanded
          ? <ChevronDown size={13} style={{ color: '#6b7280' }} />
          : <ChevronRight size={13} style={{ color: '#6b7280' }} />
        }
      </button>

      {expanded && (
        <div style={{ borderTop: '1px solid rgba(255,255,255,0.05)' }}>
          {category.events.map((evt, i) => {
            const synced = SYNC_WORTHY.has(evt.type)
            return (
              <div
                key={evt.type}
                style={{
                  display: 'flex', alignItems: 'center', gap: 10,
                  padding: '7px 14px',
                  borderBottom: i < category.events.length - 1 ? '1px solid rgba(255,255,255,0.04)' : 'none',
                  background: 'rgba(0,0,0,0.1)',
                }}
              >
                <div style={{
                  width: 6, height: 6, borderRadius: '50%', flexShrink: 0,
                  background: synced ? '#22d3ee' : '#374151',
                  boxShadow: synced ? '0 0 4px rgba(34,211,238,0.4)' : 'none',
                }} />
                <span style={{ flex: 1, fontSize: 12, color: '#d1d5db' }}>
                  {evt.label}
                </span>
                {evt.fields && (
                  <span className="font-[family-name:var(--font-mono)]" style={{ fontSize: 10, color: '#374151' }}>
                    {evt.fields}
                  </span>
                )}
                <SyncBadge synced={synced} />
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}

// ─── Main component ───────────────────────────────────────────────────────────

function About() {
  const [expandedLog, setExpandedLog] = useState({})

  const toggleLog = (id) => setExpandedLog(p => ({ ...p, [id]: !p[id] }))

  const totalTracked = EVENT_CATEGORIES.reduce((n, c) => n + c.events.length, 0)
  const totalSynced = EVENT_CATEGORIES.reduce(
    (n, c) => n + c.events.filter(e => SYNC_WORTHY.has(e.type)).length, 0
  )

  return (
    <div style={{ maxWidth: 720, margin: '0 auto', padding: '0 0 32px' }}>

      {/* Header */}
      <div style={{ padding: '20px 0 24px', borderBottom: '1px solid rgba(255,255,255,0.06)', marginBottom: 24 }}>
        <h2
          className="font-[family-name:var(--font-display)] tracking-[0.12em] uppercase text-white"
          style={{ fontSize: 14, marginBottom: 6 }}
        >
          About SC Bridge Companion
        </h2>
        <p style={{ fontSize: 12, color: '#4b5563', lineHeight: 1.6, maxWidth: 560 }}>
          SC Bridge Companion reads your local <span className="font-[family-name:var(--font-mono)]" style={{ color: '#6b7280' }}>Game.log</span> file to extract structured events.
          This page documents everything Star Citizen writes to that file, what the companion actually reads,
          and what gets synced to SC Bridge if you choose to connect.
        </p>
      </div>

      {/* Stats row */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 10, marginBottom: 28 }}>
        {[
          { label: 'Event Types Tracked', value: totalTracked },
          { label: 'Synced to SC Bridge', value: totalSynced },
          { label: 'Local Only', value: totalTracked - totalSynced },
        ].map(stat => (
          <div key={stat.label} style={{
            padding: '14px 16px',
            background: 'rgba(255,255,255,0.03)',
            border: '1px solid rgba(255,255,255,0.06)',
            borderRadius: 10,
          }}>
            <div className="font-[family-name:var(--font-mono)]" style={{ fontSize: 22, color: '#fff', marginBottom: 4 }}>
              {stat.value}
            </div>
            <div className="font-[family-name:var(--font-display)]" style={{
              fontSize: 10, letterSpacing: '0.05em', textTransform: 'uppercase', color: '#6b7280',
            }}>
              {stat.label}
            </div>
          </div>
        ))}
      </div>

      {/* Section 1: Log file contents */}
      <div style={{ marginBottom: 28 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 12 }}>
          <Tag size={13} style={{ color: '#6b7280' }} />
          <h3 className="font-[family-name:var(--font-display)]" style={{
            fontSize: 11, letterSpacing: '0.06em', textTransform: 'uppercase', color: '#6b7280',
          }}>
            What Lives in Game.log
          </h3>
        </div>

        {/* Legend */}
        <div style={{
          display: 'flex', gap: 16, marginBottom: 12,
          padding: '8px 12px',
          background: 'rgba(255,255,255,0.02)',
          border: '1px solid rgba(255,255,255,0.05)',
          borderRadius: 8,
        }}>
          {[
            { icon: <Eye size={12} style={{ color: '#6b7280' }} />, label: 'Read by companion' },
            { icon: <EyeOff size={12} style={{ color: '#374151' }} />, label: 'In log, not read' },
            { icon: <SyncBadge synced />, label: 'Sent to SC Bridge' },
          ].map((item, i) => (
            <div key={i} style={{ display: 'flex', alignItems: 'center', gap: 5 }}>
              {item.icon}
              <span style={{ fontSize: 11, color: '#4b5563' }}>{item.label}</span>
            </div>
          ))}
        </div>

        <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
          {LOG_CONTENTS.map(section => (
            <LogSection
              key={section.id}
              section={section}
              expanded={!!expandedLog[section.id]}
              onToggle={() => toggleLog(section.id)}
            />
          ))}
        </div>
      </div>

      {/* Section 2: Tracked events */}
      <div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 12 }}>
          <Shield size={13} style={{ color: '#6b7280' }} />
          <h3 className="font-[family-name:var(--font-display)]" style={{
            fontSize: 11, letterSpacing: '0.06em', textTransform: 'uppercase', color: '#6b7280',
          }}>
            Events Tracked by Companion
          </h3>
        </div>

        {/* Legend */}
        <div style={{
          display: 'flex', gap: 16, marginBottom: 12,
          padding: '8px 12px',
          background: 'rgba(255,255,255,0.02)',
          border: '1px solid rgba(255,255,255,0.05)',
          borderRadius: 8,
        }}>
          {[
            {
              icon: <div style={{ width: 6, height: 6, borderRadius: '50%', background: '#22d3ee', boxShadow: '0 0 4px rgba(34,211,238,0.4)' }} />,
              label: 'Synced to SC Bridge (when connected)',
            },
            {
              icon: <div style={{ width: 6, height: 6, borderRadius: '50%', background: '#374151' }} />,
              label: 'Local event feed only',
            },
          ].map((item, i) => (
            <div key={i} style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
              {item.icon}
              <span style={{ fontSize: 11, color: '#4b5563' }}>{item.label}</span>
            </div>
          ))}
        </div>

        <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
          {EVENT_CATEGORIES.map(cat => (
            <EventCategoryCard key={cat.name} category={cat} />
          ))}
        </div>
      </div>

      {/* Privacy note */}
      <div style={{
        marginTop: 24, padding: '12px 16px',
        background: 'rgba(34,211,238,0.04)',
        border: '1px solid rgba(34,211,238,0.1)',
        borderRadius: 10,
      }}>
        <p style={{ fontSize: 11, color: '#4b5563', lineHeight: 1.6 }}>
          <span style={{ color: '#22d3ee' }}>Privacy: </span>
          This app tracks your <strong style={{ color: '#9ca3af' }}>in-game character</strong> — handle and GEID — not you as a person. No real-world identity, hardware specs, network addresses, input device GUIDs, audio endpoints, overlay software, or Windows username are ever read or transmitted.
          Sync is opt-in — nothing leaves your machine unless you connect to SC Bridge in Settings.
        </p>
      </div>

    </div>
  )
}

export default About
