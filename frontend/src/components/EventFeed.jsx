import { useRef, useEffect, useState } from 'react'
import { Terminal, Filter } from 'lucide-react'

const SOURCE_COLORS = {
  log: 'text-sc-accent2',
}

const TYPE_COLORS = {
  // Session
  'player_login': 'text-green-400',
  'server_joined': 'text-green-400/60',
  // Ships
  'ship_boarded': 'text-sky-400',
  'ship_exited': 'text-sky-400/60',
  'hangar_ready': 'text-sky-300',
  'ship_list_fetched': 'text-sky-400/60',
  'ships_loaded': 'text-sky-400/60',
  'insurance_claim': 'text-orange-400',
  'insurance_claim_complete': 'text-orange-300',
  'vehicle_impounded': 'text-red-300',
  'fatal_collision': 'text-red-500',
  // Missions
  'contract_accepted': 'text-purple-400',
  'contract_completed': 'text-green-400',
  'contract_failed': 'text-red-400',
  'contract_available': 'text-purple-400/60',
  'new_objective': 'text-purple-300',
  'mission_ended': 'text-amber-400',
  'end_mission': 'text-amber-300',
  // Location
  'location_change': 'text-amber-400',
  'jurisdiction_entered': 'text-amber-300',
  'armistice_entered': 'text-cyan-400/60',
  'armistice_exited': 'text-cyan-400/60',
  'monitored_space_entered': 'text-gray-500',
  'monitored_space_exited': 'text-gray-500',
  // Quantum travel
  'qt_target_selected': 'text-indigo-400',
  'qt_destination_selected': 'text-indigo-400',
  'qt_fuel_requested': 'text-indigo-400/60',
  'qt_arrived': 'text-indigo-300',
  // Economy
  'money_sent': 'text-yellow-400',
  'fined': 'text-red-400',
  'transaction_complete': 'text-yellow-300',
  'rewards_earned': 'text-yellow-400',
  'refinery_complete': 'text-amber-300',
  'blueprint_received': 'text-teal-400',
  // Combat / health
  'injury': 'text-red-400',
  'incapacitated': 'text-red-500',
  'crimestat_increased': 'text-red-500',
  'emergency_services': 'text-red-300',
  // System
  'entitlement_reconciliation': 'text-gray-400',
}

function EventFeed({ events }) {
  const bottomRef = useRef(null)
  const [autoScroll, setAutoScroll] = useState(true)
  const [filter, setFilter] = useState('')

  useEffect(() => {
    if (autoScroll && bottomRef.current) {
      bottomRef.current.scrollIntoView({ behavior: 'smooth' })
    }
  }, [events, autoScroll])

  const filtered = filter
    ? events.filter(e => e.type.includes(filter) || e.source.includes(filter))
    : events

  return (
    <div className="flex flex-col" style={{ height: '100%', maxWidth: 720, margin: '0 auto' }}>
      {/* Toolbar */}
      <div className="flex items-center gap-2" style={{ marginBottom: 10 }}>
        <Terminal size={14} className="text-sc-accent" />
        <span className="font-[family-name:var(--font-display)] text-xs tracking-wider text-gray-400 uppercase">
          Live Events
        </span>
        <div className="flex-1" />

        <div style={{ position: 'relative' }}>
          <Filter size={12} style={{ position: 'absolute', left: 10, top: '50%', transform: 'translateY(-50%)', color: '#4b5563', pointerEvents: 'none' }} />
          <input
            type="text"
            placeholder="Filter..."
            value={filter}
            onChange={e => setFilter(e.target.value)}
            style={{
              width: 160, paddingLeft: 30, paddingRight: 12, paddingTop: 6, paddingBottom: 6,
              fontSize: 12, fontFamily: 'var(--font-mono)',
              background: 'rgba(255,255,255,0.03)', border: '1px solid rgba(255,255,255,0.06)',
              borderRadius: 8, color: '#d1d5db', outline: 'none',
            }}
          />
        </div>

        <button
          onClick={() => setAutoScroll(v => !v)}
          className="cursor-pointer"
          style={{
            padding: '5px 10px',
            fontSize: 11,
            borderRadius: 6,
            border: autoScroll ? '1px solid rgba(34,211,238,0.2)' : '1px solid rgba(255,255,255,0.06)',
            background: autoScroll ? 'rgba(34,211,238,0.1)' : 'transparent',
            color: autoScroll ? '#22d3ee' : '#6b7280',
            cursor: 'pointer',
            transition: 'all 0.2s',
          }}
        >
          Auto-scroll
        </button>
      </div>

      {/* Event list */}
      <div className="flex-1 overflow-y-auto bg-white/[0.02] border border-white/[0.06] rounded-xl font-[family-name:var(--font-mono)] text-xs" style={{ minHeight: 0 }}>
        {filtered.length === 0 ? (
          <div className="flex items-center justify-center h-full text-gray-600">
            {events.length === 0 ? 'Waiting for events...' : 'No matching events'}
          </div>
        ) : (
          <div className="divide-y divide-white/[0.03]">
            {filtered.map((evt, i) => (
              <div
                key={i}
                className="flex items-start gap-3 px-3 py-1.5 hover:bg-white/[0.02] transition-colors animate-fade-in"
              >
                <span className="text-gray-600 shrink-0 w-20 pt-0.5">
                  {evt.timestamp}
                </span>
                <span className={`shrink-0 w-10 uppercase text-[10px] pt-0.5 ${SOURCE_COLORS[evt.source] || 'text-gray-500'}`}>
                  {evt.source}
                </span>
                <span className={`shrink-0 ${TYPE_COLORS[evt.type] || 'text-gray-300'}`}>
                  {evt.type}
                </span>
                <span className="text-gray-600 truncate">
                  {formatData(evt.data)}
                </span>
              </div>
            ))}
            <div ref={bottomRef} />
          </div>
        )}
      </div>

      {/* Footer stats */}
      <div className="flex items-center justify-between text-[10px] text-gray-600 font-[family-name:var(--font-mono)]" style={{ marginTop: 6 }}>
        <span>{filtered.length} events{filter ? ` (${events.length} total)` : ''}</span>
        <span>Buffer: {events.length}/200</span>
      </div>
    </div>
  )
}

function formatData(data) {
  if (!data) return ''
  const entries = Object.entries(data).filter(([k]) => k !== 'direction' && k !== 'method')
  if (entries.length === 0) return ''
  return entries.map(([k, v]) => `${k}=${v}`).join(' ')
}

export default EventFeed
