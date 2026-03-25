import { FileText, Database, Globe, ExternalLink } from 'lucide-react'

const wails = window.go?.main?.App

const EXTENSIONS = [
  {
    name: 'Chrome',
    url: 'https://chromewebstore.google.com/detail/sc-bridge-sync/gcokkoamjodagagbojhkimfbjjpdfefi',
    icon: (
      <svg viewBox="0 0 24 24" width="22" height="22" fill="#4285F4" xmlns="http://www.w3.org/2000/svg"><path d="M12 0C8.21 0 4.831 1.757 2.632 4.501l3.953 6.848A5.454 5.454 0 0 1 12 6.545h10.691A12 12 0 0 0 12 0zM1.931 5.47A11.943 11.943 0 0 0 0 12c0 6.012 4.42 10.991 10.189 11.864l3.953-6.847a5.45 5.45 0 0 1-6.865-2.29zm13.342 2.166a5.446 5.446 0 0 1 1.45 7.09l.002.001h-.002l-5.344 9.257c.206.01.413.016.621.016 6.627 0 12-5.373 12-12 0-1.54-.29-3.011-.818-4.364zM12 16.364a4.364 4.364 0 1 1 0-8.728 4.364 4.364 0 0 1 0 8.728Z"/></svg>
    ),
    color: '#4285f4',
  },
  {
    name: 'Firefox',
    url: 'https://addons.mozilla.org/en-US/firefox/addon/sc-bridge-sync/',
    icon: (
      <svg viewBox="0 0 24 24" width="22" height="22" fill="#FF7139" xmlns="http://www.w3.org/2000/svg"><path d="M20.452 3.445a11.002 11.002 0 00-2.482-1.908C16.944.997 15.098.093 12.477.032c-.734-.017-1.457.03-2.174.144-.72.114-1.398.292-2.118.56-1.017.377-1.996.975-2.574 1.554.583-.349 1.476-.733 2.55-.992a10.083 10.083 0 013.729-.167c2.341.34 4.178 1.381 5.48 2.625a8.066 8.066 0 011.298 1.587c1.468 2.382 1.33 5.376.184 7.142-.85 1.312-2.67 2.544-4.37 2.53-.583-.023-1.438-.152-2.25-.566-2.629-1.343-3.021-4.688-1.118-6.306-.632-.136-1.82.13-2.646 1.363-.742 1.107-.7 2.816-.242 4.028a6.473 6.473 0 01-.59-1.895 7.695 7.695 0 01.416-3.845A8.212 8.212 0 019.45 5.399c.896-1.069 1.908-1.72 2.75-2.005-.54-.471-1.411-.738-2.421-.767C8.31 2.583 6.327 3.061 4.7 4.41a8.148 8.148 0 00-1.976 2.414c-.455.836-.691 1.659-.697 1.678.122-1.445.704-2.994 1.248-4.055-.79.413-1.827 1.668-2.41 3.042C.095 9.37-.2 11.608.14 13.989c.966 5.668 5.9 9.982 11.843 9.982C18.62 23.971 24 18.591 24 11.956a11.93 11.93 0 00-3.548-8.511z"/></svg>
    ),
    color: '#ff7139',
  },
  {
    name: 'Edge',
    url: 'https://microsoftedge.microsoft.com/addons/detail/sc-bridge-sync/edndedmmbdbofdphimpcofdccbpbgjib',
    icon: (
      <svg viewBox="0 0 24 24" width="22" height="22" fill="#0078D4" xmlns="http://www.w3.org/2000/svg"><path d="M21.86 17.86q.14 0 .25.12.1.13.1.25t-.11.33l-.32.46-.43.53-.44.5q-.21.25-.38.42l-.22.23q-.58.53-1.34 1.04-.76.51-1.6.91-.86.4-1.74.64t-1.67.24q-.9 0-1.69-.28-.8-.28-1.48-.78-.68-.5-1.22-1.17-.53-.66-.92-1.44-.38-.77-.58-1.6-.2-.83-.2-1.67 0-1 .32-1.96.33-.97.87-1.8.14.95.55 1.77.41.82 1.02 1.5.6.68 1.38 1.21.78.54 1.64.9.86.36 1.77.56.92.2 1.8.2 1.12 0 2.18-.24 1.06-.23 2.06-.72l.2-.1.2-.05zm-15.5-1.27q0 1.1.27 2.15.27 1.06.78 2.03.51.96 1.24 1.77.74.82 1.66 1.4-1.47-.2-2.8-.74-1.33-.55-2.48-1.37-1.15-.83-2.08-1.9-.92-1.07-1.58-2.33T.36 14.94Q0 13.54 0 12.06q0-.81.32-1.49.31-.68.83-1.23.53-.55 1.2-.96.66-.4 1.35-.66.74-.27 1.5-.39.78-.12 1.55-.12.7 0 1.42.1.72.12 1.4.35.68.23 1.32.57.63.35 1.16.83-.35 0-.7.07-.33.07-.65.23v-.02q-.63.28-1.2.74-.57.46-1.05 1.04-.48.58-.87 1.26-.38.67-.65 1.39-.27.71-.42 1.44-.15.72-.15 1.38zM11.96.06q1.7 0 3.33.39 1.63.38 3.07 1.15 1.43.77 2.62 1.93 1.18 1.16 1.98 2.7.49.94.76 1.96.28 1 .28 2.08 0 .89-.23 1.7-.24.8-.69 1.48-.45.68-1.1 1.22-.64.53-1.45.88-.54.24-1.11.36-.58.13-1.16.13-.42 0-.97-.03-.54-.03-1.1-.12-.55-.1-1.05-.28-.5-.19-.84-.5-.12-.09-.23-.24-.1-.16-.1-.33 0-.15.16-.35.16-.2.35-.5.2-.28.36-.68.16-.4.16-.95 0-1.06-.4-1.96-.4-.91-1.06-1.64-.66-.74-1.52-1.28-.86-.55-1.79-.89-.84-.3-1.72-.44-.87-.14-1.76-.14-1.55 0-3.06.45T.94 7.55q.71-1.74 1.81-3.13 1.1-1.38 2.52-2.35Q6.68 1.1 8.37.58q1.7-.52 3.58-.52Z"/></svg>
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
            : status?.tailerActive
              ? 'Monitoring — waiting for player login...'
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
