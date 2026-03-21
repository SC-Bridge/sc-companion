import { FileText, Database, Globe, ExternalLink } from 'lucide-react'

const wails = window.go?.main?.App

const EXTENSIONS = [
  {
    name: 'Chrome',
    url: 'https://chromewebstore.google.com/detail/sc-bridge-sync/gcokkoamjodagagbojhkimfbjjpdfefi',
    icon: (
      <svg viewBox="0 0 24 24" width="18" height="18" fill="none">
        <circle cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="1.5" opacity="0.3" />
        <circle cx="12" cy="12" r="4" stroke="currentColor" strokeWidth="1.5" />
        <line x1="12" y1="8" x2="21" y2="5" stroke="currentColor" strokeWidth="1.5" opacity="0.6" />
        <line x1="8.5" y1="14" x2="3" y2="18" stroke="currentColor" strokeWidth="1.5" opacity="0.6" />
        <line x1="15.5" y1="14" x2="21" y2="18" stroke="currentColor" strokeWidth="1.5" opacity="0.6" />
      </svg>
    ),
    color: '#4285f4',
  },
  {
    name: 'Firefox',
    url: 'https://addons.mozilla.org/en-US/firefox/addon/sc-bridge-sync/',
    icon: (
      <svg viewBox="0 0 24 24" width="18" height="18" fill="none">
        <circle cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="1.5" opacity="0.3" />
        <path d="M7 8c1-3 4-4 5-4s3 .5 4 2c1 1.5.5 3 .5 3s1-1.5-1-3c-1.5-1.2-3-1-3-1s2.5.5 3 3c.3 1.5-.5 3-2 4s-3.5 1-5 0-2.5-3-1.5-4z" stroke="currentColor" strokeWidth="1.2" />
      </svg>
    ),
    color: '#ff7139',
  },
  {
    name: 'Edge',
    url: 'https://microsoftedge.microsoft.com/addons/detail/sc-bridge-sync/edndedmmbdbofdphimpcofdccbpbgjib',
    icon: (
      <svg viewBox="0 0 24 24" width="18" height="18" fill="none">
        <path d="M12 2C6.5 2 2 6.5 2 12s4.5 10 10 10c4.3 0 7.9-2.7 9.3-6.5" stroke="currentColor" strokeWidth="1.5" opacity="0.3" />
        <path d="M20 8c-1-2.5-3-4.5-5.5-5.5M8 20c1.5-2 4-6 4-8s-1-3.5-1-3.5c2 0 4.5 1.5 5 4s-1 5-4 6.5" stroke="currentColor" strokeWidth="1.5" />
      </svg>
    ),
    color: '#0078d7',
  },
]

const StatCard = ({ icon: Icon, label, value, variant }) => (
  <div style={{
    padding: '16px 18px',
    background: 'rgba(255,255,255,0.03)',
    border: '1px solid rgba(255,255,255,0.06)',
    borderRadius: 12,
    display: 'flex',
    alignItems: 'center',
    gap: 14,
  }}>
    <div style={{
      width: 32, height: 32, borderRadius: 8,
      background: variant === 'success' ? 'rgba(46,196,182,0.1)' : 'rgba(34,211,238,0.1)',
      display: 'flex', alignItems: 'center', justifyContent: 'center',
      flexShrink: 0,
    }}>
      <Icon size={16} style={{ color: variant === 'success' ? '#2ec4b6' : '#22d3ee' }} />
    </div>
    <div style={{ minWidth: 0 }}>
      <div className="font-[family-name:var(--font-display)]" style={{ fontSize: 11, letterSpacing: '0.05em', color: '#6b7280', textTransform: 'uppercase' }}>
        {label}
      </div>
      <div className="font-[family-name:var(--font-mono)]" style={{ fontSize: 18, color: '#fff', marginTop: 2 }}>
        {value}
      </div>
    </div>
  </div>
)

function Dashboard({ status, config }) {
  return (
    <div style={{ maxWidth: 720, margin: '0 auto' }}>
      {/* Hero */}
      <div className="text-center" style={{ paddingTop: 12, paddingBottom: 28 }}>
        <div style={{ marginBottom: 16 }}>
          <svg viewBox="0 0 100 100" style={{ width: 80, height: 80, margin: '0 auto' }}>
            <circle cx="50" cy="50" r="42" fill="none" stroke="rgba(34, 211, 238, 0.15)" strokeWidth="2" />
            <path d="M 15 50 A 35 35 0 0 1 85 50" fill="none" stroke="rgba(34, 211, 238, 0.6)" strokeWidth="6" strokeLinecap="round" strokeDasharray="40 12" style={{ filter: 'drop-shadow(0 0 4px rgba(34, 211, 238, 0.4))' }} />
            <path d="M 85 50 A 35 35 0 0 1 15 50" fill="none" stroke="rgba(34, 211, 238, 0.6)" strokeWidth="6" strokeLinecap="round" strokeDasharray="40 12" style={{ filter: 'drop-shadow(0 0 4px rgba(34, 211, 238, 0.4))' }} />
            <line x1="8" y1="50" x2="92" y2="50" stroke="rgba(34, 211, 238, 0.8)" strokeWidth="5" strokeLinecap="round" style={{ filter: 'drop-shadow(0 0 6px rgba(34, 211, 238, 0.5))' }} />
          </svg>
        </div>
        <h1
          className="font-[family-name:var(--font-display)] text-xl text-white tracking-[0.15em] uppercase"
          style={{ textShadow: '0 0 30px rgba(34, 211, 238, 0.2)' }}
        >
          SC Bridge Companion
        </h1>
        <p className="text-gray-500 text-sm mt-1.5 tracking-wide">
          {status?.playerHandle
            ? `Welcome back, ${status.playerHandle}`
            : 'Waiting for Star Citizen...'
          }
        </p>
      </div>

      {/* Stats grid */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: 12, marginBottom: 20 }}>
        <StatCard
          icon={FileText}
          label="Game.log"
          value={status?.tailerActive ? 'Tailing' : 'Disconnected'}
          variant={status?.tailerActive ? 'success' : undefined}
        />
        <StatCard
          icon={Database}
          label="Events"
          value={status?.eventCount || 0}
        />
      </div>

      {/* Data sources */}
      <div style={{ padding: '18px 20px', background: 'rgba(255,255,255,0.03)', border: '1px solid rgba(255,255,255,0.06)', borderRadius: 12 }}>
        <h3 className="font-[family-name:var(--font-display)]" style={{ fontSize: 11, letterSpacing: '0.05em', color: '#6b7280', textTransform: 'uppercase', marginBottom: 16 }}>
          Data Sources
        </h3>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
          <DataSourceRow
            label="Game.log Tailer"
            description="Parses in-game events: ship boarding, contracts, locations, economy, missions, quantum travel"
            active={status?.tailerActive}
          />
          <DataSourceRow
            label="SC Bridge Sync"
            description="Uploads game events to scbridge.app for fleet tracking and analysis"
            active={config?.connected}
            note={config?.connected ? undefined : 'Connect in Settings'}
          />
        </div>
      </div>

      {/* Browser Extensions */}
      <div style={{ padding: '18px 20px', background: 'rgba(255,255,255,0.03)', border: '1px solid rgba(255,255,255,0.06)', borderRadius: 12, marginTop: 16 }}>
        <h3 className="font-[family-name:var(--font-display)]" style={{ fontSize: 11, letterSpacing: '0.05em', color: '#6b7280', textTransform: 'uppercase', marginBottom: 6 }}>
          Browser Extension
        </h3>
        <p style={{ fontSize: 12, color: '#4b5563', marginBottom: 14, lineHeight: 1.5 }}>
          Sync your hangar, pledges, and buyback data directly from the RSI website
        </p>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 10 }}>
          {EXTENSIONS.map(ext => (
            <button
              key={ext.name}
              onClick={() => wails?.OpenDownloadURL(ext.url)}
              style={{
                display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 8,
                padding: '14px 12px',
                background: 'rgba(255,255,255,0.02)',
                border: '1px solid rgba(255,255,255,0.06)',
                borderRadius: 10,
                cursor: 'pointer',
                transition: 'all 200ms',
              }}
              onMouseEnter={e => {
                e.currentTarget.style.background = `${ext.color}15`
                e.currentTarget.style.borderColor = `${ext.color}40`
              }}
              onMouseLeave={e => {
                e.currentTarget.style.background = 'rgba(255,255,255,0.02)'
                e.currentTarget.style.borderColor = 'rgba(255,255,255,0.06)'
              }}
            >
              <div style={{ color: ext.color, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                {ext.icon}
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                <span style={{ fontSize: 12, color: '#d1d5db', fontWeight: 500 }}>{ext.name}</span>
                <ExternalLink size={10} style={{ color: '#6b7280' }} />
              </div>
            </button>
          ))}
        </div>
      </div>
    </div>
  )
}

const DataSourceRow = ({ label, description, active, note }) => (
  <div style={{ display: 'flex', alignItems: 'flex-start', gap: 12 }}>
    <div style={{
      width: 8, height: 8, borderRadius: '50%', marginTop: 6, flexShrink: 0,
      background: active ? '#2ec4b6' : '#4b5563',
      boxShadow: active ? '0 0 6px rgba(46,196,182,0.5)' : 'none',
    }} />
    <div style={{ minWidth: 0 }}>
      <div style={{ fontSize: 13, color: '#d1d5db' }}>{label}</div>
      <div style={{ fontSize: 12, color: '#4b5563', marginTop: 2, lineHeight: 1.5 }}>{description}</div>
      {note && !active && (
        <div style={{ fontSize: 12, color: 'rgba(245,166,35,0.6)', marginTop: 3 }}>{note}</div>
      )}
    </div>
  </div>
)

export default Dashboard
