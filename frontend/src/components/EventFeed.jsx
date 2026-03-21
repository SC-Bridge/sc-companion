import { useRef, useEffect, useState } from 'react'
import { Terminal, Filter, Trash2 } from 'lucide-react'

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
    <div className="flex flex-col h-full max-w-4xl mx-auto">
      {/* Toolbar */}
      <div className="flex items-center gap-2 mb-3">
        <Terminal size={14} className="text-sc-accent" />
        <span className="font-[family-name:var(--font-display)] text-xs tracking-wider text-gray-400 uppercase">
          Live Event Feed
        </span>
        <div className="flex-1" />

        <div className="relative">
          <Filter size={12} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-gray-600" />
          <input
            type="text"
            placeholder="Filter events..."
            value={filter}
            onChange={e => setFilter(e.target.value)}
            className="w-48 pl-7 pr-3 py-1.5 text-xs font-[family-name:var(--font-mono)] bg-white/[0.03] border border-white/[0.06] rounded-lg text-gray-300 placeholder-gray-600 focus:outline-none focus:border-sc-accent/30"
          />
        </div>

        <button
          onClick={() => setAutoScroll(v => !v)}
          className={`px-2.5 py-1.5 text-xs rounded-lg border transition-colors ${
            autoScroll
              ? 'bg-sc-accent/10 text-sc-accent border-sc-accent/20'
              : 'text-gray-500 border-white/[0.06] hover:text-gray-300'
          }`}
        >
          Auto-scroll
        </button>
      </div>

      {/* Event list */}
      <div className="flex-1 overflow-y-auto bg-white/[0.02] border border-white/[0.06] rounded-xl font-[family-name:var(--font-mono)] text-xs">
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
      <div className="flex items-center justify-between mt-2 text-[10px] text-gray-600 font-[family-name:var(--font-mono)]">
        <span>{filtered.length} events{filter ? ` (${events.length} total)` : ''}</span>
        <span>Buffer: {events.length}/{200}</span>
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
