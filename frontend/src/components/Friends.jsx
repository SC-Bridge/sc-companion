import { useState, useEffect, useRef } from 'react'
import { Users, Search, Circle } from 'lucide-react'

const wails = window.go?.main?.App

const PRESENCE_ORDER = { online: 0, away: 1, offline: 2 }
const PRESENCE_COLORS = {
  online: '#2ec4b6',
  away: '#f59e0b',
  offline: '#4b5563',
}
const PRESENCE_LABELS = {
  online: 'Online',
  away: 'Away',
  offline: 'Offline',
}

function Friends({ config }) {
  const [friends, setFriends] = useState([])
  const [filter, setFilter] = useState('')
  const [loading, setLoading] = useState(true)
  const lastPollRef = useRef(null)

  // Initial full load via Go backend
  useEffect(() => {
    if (!wails || !config?.connected) {
      setLoading(false)
      return
    }

    async function loadFriends() {
      try {
        const result = await wails.GetFriends()
        if (result) {
          setFriends(result)
          lastPollRef.current = new Date().toISOString()
        }
      } catch (e) {
        console.error('Friends load failed:', e)
      }
      setLoading(false)
    }

    loadFriends()
  }, [config?.connected])

  // Delta polling every 30s via Go backend
  useEffect(() => {
    if (!wails || !config?.connected) return

    const interval = setInterval(async () => {
      if (!lastPollRef.current) return
      try {
        const delta = await wails.GetFriendsDelta(lastPollRef.current)
        if (delta && delta.length > 0) {
          setFriends(prev => {
            const updated = new Map(prev.map(f => [f.account_id, f]))
            for (const f of delta) {
              updated.set(f.account_id, f)
            }
            return Array.from(updated.values())
          })
        }
        lastPollRef.current = new Date().toISOString()
      } catch (e) {
        console.error('Friends delta poll failed:', e)
      }
    }, 30000)

    return () => clearInterval(interval)
  }, [config?.connected])

  // Sort: online first, then away, then offline. Alpha within each group.
  const sorted = [...friends].sort((a, b) => {
    const pa = PRESENCE_ORDER[a.presence] ?? 2
    const pb = PRESENCE_ORDER[b.presence] ?? 2
    if (pa !== pb) return pa - pb
    return (a.display_name || a.nickname || '').localeCompare(b.display_name || b.nickname || '')
  })

  const filtered = filter
    ? sorted.filter(f =>
        (f.display_name || '').toLowerCase().includes(filter.toLowerCase()) ||
        (f.nickname || '').toLowerCase().includes(filter.toLowerCase())
      )
    : sorted

  const onlineCount = friends.filter(f => f.presence === 'online').length
  const awayCount = friends.filter(f => f.presence === 'away').length

  if (!config?.connected) {
    return (
      <div style={{ maxWidth: 720, margin: '0 auto' }}>
        <div className="flex items-center gap-2" style={{ marginBottom: 16 }}>
          <Users size={16} className="text-sc-accent" />
          <h2 className="font-[family-name:var(--font-display)] text-sm tracking-wider text-gray-400 uppercase">
            Friends
          </h2>
        </div>
        <div style={{
          padding: '40px 20px',
          background: 'rgba(255,255,255,0.03)',
          border: '1px solid rgba(255,255,255,0.06)',
          borderRadius: 12,
          textAlign: 'center',
        }}>
          <p className="text-gray-500 text-sm">Connect to SC Bridge in Settings to see your friends list.</p>
        </div>
      </div>
    )
  }

  return (
    <div style={{ maxWidth: 720, margin: '0 auto' }}>
      {/* Header */}
      <div className="flex items-center gap-2" style={{ marginBottom: 12 }}>
        <Users size={16} className="text-sc-accent" />
        <h2 className="font-[family-name:var(--font-display)] text-sm tracking-wider text-gray-400 uppercase">
          Friends
        </h2>
        <div className="flex-1" />

        {/* Presence summary */}
        <div className="flex items-center gap-3 font-[family-name:var(--font-mono)] text-xs">
          {onlineCount > 0 && (
            <span className="flex items-center gap-1.5">
              <Circle size={7} fill="#2ec4b6" stroke="none" />
              <span style={{ color: '#2ec4b6' }}>{onlineCount}</span>
            </span>
          )}
          {awayCount > 0 && (
            <span className="flex items-center gap-1.5">
              <Circle size={7} fill="#f59e0b" stroke="none" />
              <span style={{ color: '#f59e0b' }}>{awayCount}</span>
            </span>
          )}
          <span className="text-gray-600">{friends.length} total</span>
        </div>
      </div>

      {/* Search */}
      <div className="relative" style={{ marginBottom: 10 }}>
        <Search size={13} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-600" />
        <input
          type="text"
          placeholder="Search friends..."
          value={filter}
          onChange={e => setFilter(e.target.value)}
          className="w-full pl-8 pr-3 py-2 text-sm font-[family-name:var(--font-body)] bg-white/[0.03] border border-white/[0.06] rounded-lg text-gray-300 placeholder-gray-600 focus:outline-none focus:border-sc-accent/30"
        />
      </div>

      {/* Friends list */}
      <div style={{
        background: 'rgba(255,255,255,0.02)',
        border: '1px solid rgba(255,255,255,0.06)',
        borderRadius: 12,
        overflow: 'hidden',
      }}>
        {loading ? (
          <div className="flex items-center justify-center text-gray-600 text-sm" style={{ padding: 40 }}>
            Loading friends...
          </div>
        ) : filtered.length === 0 ? (
          <div className="flex items-center justify-center text-gray-600 text-sm" style={{ padding: 40 }}>
            {friends.length === 0 ? 'No friends data yet. Sync via the browser extension.' : 'No matching friends'}
          </div>
        ) : (
          <div>
            {filtered.map((friend) => (
              <FriendRow key={friend.account_id} friend={friend} />
            ))}
          </div>
        )}
      </div>

      {/* Footer */}
      <div className="flex items-center justify-between font-[family-name:var(--font-mono)] text-[10px] text-gray-600" style={{ marginTop: 6 }}>
        <span>{filtered.length} shown{filter ? ` of ${friends.length}` : ''}</span>
        <span>Updates every 30s</span>
      </div>
    </div>
  )
}

function FriendRow({ friend }) {
  const presence = friend.presence || 'offline'
  const color = PRESENCE_COLORS[presence] || PRESENCE_COLORS.offline
  const name = friend.display_name || friend.nickname || 'Unknown'
  const handle = friend.nickname || ''

  return (
    <div
      className="flex items-center gap-3 hover:bg-white/[0.02] transition-colors"
      style={{
        padding: '8px 14px',
        borderBottom: '1px solid rgba(255,255,255,0.03)',
      }}
    >
      {/* Presence dot */}
      <div style={{
        width: 8, height: 8, borderRadius: '50%',
        background: color,
        boxShadow: presence === 'online' ? `0 0 6px ${color}80` : 'none',
        flexShrink: 0,
      }} />

      {/* Name + handle */}
      <div style={{ flex: 1, minWidth: 0 }}>
        <div className="text-sm text-gray-200 truncate">{name}</div>
        {handle && handle !== name && (
          <div className="text-xs text-gray-600 font-[family-name:var(--font-mono)] truncate">{handle}</div>
        )}
      </div>

      {/* Presence label */}
      <span
        className="font-[family-name:var(--font-mono)] text-xs shrink-0"
        style={{ color, opacity: presence === 'offline' ? 0.5 : 0.8 }}
      >
        {PRESENCE_LABELS[presence] || presence}
      </span>
    </div>
  )
}

export default Friends
