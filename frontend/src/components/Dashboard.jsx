import { Radio, FileText, Database, Shield, Zap, RefreshCw } from 'lucide-react'

const StatCard = ({ icon: Icon, label, value }) => (
  <div
    style={{
      padding: 20,
      background: 'rgba(255,255,255,0.03)',
      border: '1px solid rgba(255,255,255,0.06)',
      borderRadius: 12,
      display: 'flex',
      alignItems: 'center',
      gap: 14,
      transition: 'border-color 0.3s',
    }}
  >
    <div style={{
      width: 32, height: 32, borderRadius: 8,
      background: 'rgba(34,211,238,0.1)',
      display: 'flex', alignItems: 'center', justifyContent: 'center',
      flexShrink: 0,
    }}>
      <Icon size={16} style={{ color: '#22d3ee' }} />
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

function Dashboard({ status }) {
  return (
    <div style={{ maxWidth: 960, margin: '0 auto', display: 'flex', flexDirection: 'column', gap: 32 }}>
      {/* Hero */}
      <div className="relative text-center py-8">
        <div className="relative inline-block mb-4">
          <svg viewBox="0 0 100 100" className="w-24 h-24 mx-auto">
            <circle
              cx="50" cy="50" r="42"
              fill="none"
              stroke="rgba(34, 211, 238, 0.15)"
              strokeWidth="2"
            />
            <path
              d="M 15 50 A 35 35 0 0 1 85 50"
              fill="none"
              stroke="rgba(34, 211, 238, 0.6)"
              strokeWidth="6"
              strokeLinecap="round"
              strokeDasharray="40 12"
              style={{ filter: 'drop-shadow(0 0 4px rgba(34, 211, 238, 0.4))' }}
            />
            <path
              d="M 85 50 A 35 35 0 0 1 15 50"
              fill="none"
              stroke="rgba(34, 211, 238, 0.6)"
              strokeWidth="6"
              strokeLinecap="round"
              strokeDasharray="40 12"
              style={{ filter: 'drop-shadow(0 0 4px rgba(34, 211, 238, 0.4))' }}
            />
            <line
              x1="8" y1="50" x2="92" y2="50"
              stroke="rgba(34, 211, 238, 0.8)"
              strokeWidth="5"
              strokeLinecap="round"
              style={{ filter: 'drop-shadow(0 0 6px rgba(34, 211, 238, 0.5))' }}
            />
          </svg>
        </div>
        <h1
          className="font-[family-name:var(--font-display)] text-2xl text-white tracking-[0.15em] uppercase"
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
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: 16 }}>
        <StatCard
          icon={Zap}
          label="CIG API"
          value={status?.gameConnected ? 'Connected' : 'Waiting'}
        />
        <StatCard
          icon={RefreshCw}
          label="Data Sync"
          value={status?.syncActive ? 'Active' : 'Inactive'}
        />
        <StatCard
          icon={FileText}
          label="Game.log"
          value={status?.tailerActive ? 'Tailing' : 'Disconnected'}
        />
        <StatCard
          icon={Database}
          label="Events"
          value={status?.eventCount || 0}
        />
      </div>

      {/* Connection info card */}
      <div style={{ padding: 24, background: 'rgba(255,255,255,0.03)', border: '1px solid rgba(255,255,255,0.06)', borderRadius: 12 }}>
        <h3 className="font-[family-name:var(--font-display)]" style={{ fontSize: 14, letterSpacing: '0.05em', color: '#9ca3af', textTransform: 'uppercase', marginBottom: 24 }}>
          Data Sources
        </h3>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 20 }}>
          <DataSourceRow
            label="Game.log Tailer"
            description="Parses in-game events: ship boarding, contracts, location changes, money transfers"
            active={status?.tailerActive}
          />
          <DataSourceRow
            label="CIG gRPC Client"
            description="Direct connection to CIG backend — wallet, friends, reputation, blueprints, entitlements, missions, stats"
            active={status?.gameConnected}
            note="Launch Star Citizen to connect"
          />
          <DataSourceRow
            label="Data Sync to SC Bridge"
            description="Syncs gRPC data to scbridge.app on a schedule (wallet 30s, friends 60s, rep 5m, entitlements 10m)"
            active={status?.syncActive}
            note="Set API token in settings"
          />
        </div>
      </div>
    </div>
  )
}

const DataSourceRow = ({ label, description, active, note }) => (
  <div style={{ display: 'flex', alignItems: 'flex-start', gap: 14, paddingLeft: 4 }}>
    <div style={{
      width: 8, height: 8, borderRadius: '50%', marginTop: 6, flexShrink: 0,
      background: active ? '#2ec4b6' : '#4b5563',
      boxShadow: active ? '0 0 6px rgba(46,196,182,0.5)' : 'none',
    }} />
    <div>
      <div style={{ fontSize: 14, color: '#d1d5db' }}>{label}</div>
      <div style={{ fontSize: 12, color: '#4b5563', marginTop: 2 }}>{description}</div>
      {note && !active && (
        <div style={{ fontSize: 12, color: 'rgba(245,166,35,0.6)', marginTop: 4 }}>{note}</div>
      )}
    </div>
  </div>
)

export default Dashboard
