import { Radio, FileText, Database, Shield } from 'lucide-react'

const StatCard = ({ icon: Icon, label, value, accent }) => (
  <div className="relative bg-white/[0.03] backdrop-blur-md border border-white/[0.06] rounded-xl p-4 group hover:border-sc-accent/20 transition-all duration-300">
    {/* HUD corners */}
    <div className="absolute -top-1 -left-1 w-4 h-4 border-t border-l border-sc-accent/0 group-hover:border-sc-accent/20 transition-colors" />
    <div className="absolute -top-1 -right-1 w-4 h-4 border-t border-r border-sc-accent/0 group-hover:border-sc-accent/20 transition-colors" />
    <div className="absolute -bottom-1 -left-1 w-4 h-4 border-b border-l border-sc-accent/0 group-hover:border-sc-accent/20 transition-colors" />
    <div className="absolute -bottom-1 -right-1 w-4 h-4 border-b border-r border-sc-accent/0 group-hover:border-sc-accent/20 transition-colors" />

    <div className="flex items-center gap-3">
      <div className={`w-9 h-9 rounded-lg flex items-center justify-center ${accent || 'bg-sc-accent/10'}`}>
        <Icon size={18} className="text-sc-accent" />
      </div>
      <div>
        <div className="text-[11px] font-[family-name:var(--font-display)] tracking-wider text-gray-500 uppercase">
          {label}
        </div>
        <div className="text-lg font-[family-name:var(--font-mono)] text-white mt-0.5">
          {value}
        </div>
      </div>
    </div>
  </div>
)

function Dashboard({ status }) {
  return (
    <div className="max-w-3xl mx-auto space-y-6">
      {/* Hero */}
      <div className="relative text-center py-8">
        <div className="relative inline-block mb-4">
          <svg viewBox="0 0 100 100" className="w-24 h-24 mx-auto">
            {/* Outer ring */}
            <circle
              cx="50" cy="50" r="42"
              fill="none"
              stroke="rgba(34, 211, 238, 0.15)"
              strokeWidth="2"
            />
            {/* Segmented arc - top */}
            <path
              d="M 15 50 A 35 35 0 0 1 85 50"
              fill="none"
              stroke="rgba(34, 211, 238, 0.6)"
              strokeWidth="6"
              strokeLinecap="round"
              strokeDasharray="40 12"
              style={{ filter: 'drop-shadow(0 0 4px rgba(34, 211, 238, 0.4))' }}
            />
            {/* Segmented arc - bottom */}
            <path
              d="M 85 50 A 35 35 0 0 1 15 50"
              fill="none"
              stroke="rgba(34, 211, 238, 0.6)"
              strokeWidth="6"
              strokeLinecap="round"
              strokeDasharray="40 12"
              style={{ filter: 'drop-shadow(0 0 4px rgba(34, 211, 238, 0.4))' }}
            />
            {/* Horizontal bar */}
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
        <p className="text-gray-500 text-sm mt-1 tracking-wide">
          {status?.playerHandle
            ? `Welcome back, ${status.playerHandle}`
            : 'Waiting for Star Citizen...'
          }
        </p>
      </div>

      {/* Stats grid */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-3">
        <StatCard
          icon={Radio}
          label="gRPC Proxy"
          value={status?.proxyRunning ? 'Active' : 'Inactive'}
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
        <StatCard
          icon={Shield}
          label="Last Event"
          value={status?.lastEvent || '—'}
        />
      </div>

      {/* Connection info card */}
      <div className="bg-white/[0.03] backdrop-blur-md border border-white/[0.06] rounded-xl p-5">
        <h3 className="font-[family-name:var(--font-display)] text-sm tracking-wider text-gray-400 uppercase mb-4">
          Data Sources
        </h3>
        <div className="space-y-3">
          <DataSourceRow
            label="Game.log Tailer"
            description="Parses in-game events: ship boarding, contracts, location changes, money transfers"
            active={status?.tailerActive}
          />
          <DataSourceRow
            label="gRPC Interceptor"
            description="Captures wallet balance, reputation, blueprints, friend presence from CIG backend"
            active={status?.proxyRunning}
          />
          <DataSourceRow
            label="API Sync"
            description="Uploads events to scbridge.app for fleet tracking and analysis"
            active={false}
            note="Set API token in settings"
          />
        </div>
      </div>
    </div>
  )
}

const DataSourceRow = ({ label, description, active, note }) => (
  <div className="flex items-start gap-3 py-2">
    <div className={`w-2 h-2 rounded-full mt-1.5 shrink-0 ${
      active ? 'bg-sc-success shadow-[0_0_6px_rgba(46,196,182,0.5)]' : 'bg-gray-600'
    }`} />
    <div>
      <div className="text-sm text-gray-300">{label}</div>
      <div className="text-xs text-gray-600 mt-0.5">{description}</div>
      {note && !active && (
        <div className="text-xs text-sc-warn/60 mt-1">{note}</div>
      )}
    </div>
  </div>
)

export default Dashboard
