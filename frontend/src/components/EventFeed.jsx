import { useRef, useEffect, useState } from 'react'
import { Terminal, Filter, Trash2 } from 'lucide-react'

const SOURCE_COLORS = {
  grpc: 'text-sc-accent',
  log: 'text-sc-accent2',
}

const TYPE_COLORS = {
  'ship_boarded': 'text-sky-400',
  'ship_exited': 'text-sky-400/60',
  'player_login': 'text-green-400',
  'location_change': 'text-amber-400',
  'contract_accepted': 'text-purple-400',
  'contract_completed': 'text-green-400',
  'money_sent': 'text-yellow-400',
  'injury': 'text-red-400',
  'incapacitated': 'text-red-500',
  'cig_connected': 'text-emerald-400',
  'grpc_sync_ok': 'text-cyan-400',
  'grpc_sync_error': 'text-red-400',
  'wallet_data': 'text-yellow-400',
  'wallet_error': 'text-red-400',
  'friends_data': 'text-blue-400',
  'friends_error': 'text-red-400',
  'reputation_data': 'text-purple-400',
  'reputation_error': 'text-red-400',
  'blueprints_data': 'text-teal-400',
  'blueprints_error': 'text-red-400',
  'entitlements_data': 'text-indigo-400',
  'entitlements_error': 'text-red-400',
  'missions_data': 'text-orange-400',
  'missions_error': 'text-red-400',
  'stats_data': 'text-lime-400',
  'stats_error': 'text-red-400',
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
