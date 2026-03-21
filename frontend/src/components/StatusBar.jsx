import { Wifi, WifiOff, User, Ship, MapPin } from 'lucide-react'

function StatusBar({ status }) {
  if (!status) return null

  return (
    <div className="app-statusbar font-[family-name:var(--font-mono)]" style={{ display: 'flex', alignItems: 'center', height: 28, borderTop: '1px solid rgba(255,255,255,0.04)', background: 'rgba(9,19,31,0.6)', fontSize: 11, color: '#6b7280', gap: 16, userSelect: 'none' }}>
      {/* Log tailer */}
      <div className="flex items-center gap-1.5">
        {status.tailerActive ? (
          <>
            <Wifi size={11} className="text-sc-success" />
            <span className="text-sc-success/80">Game.log</span>
          </>
        ) : (
          <>
            <WifiOff size={11} />
            <span>No Log</span>
          </>
        )}
      </div>

      <div className="w-px h-3 bg-white/[0.06]" />

      {/* Player info */}
      {status.playerHandle && (
        <div className="flex items-center gap-1.5">
          <User size={11} className="text-sc-accent2" />
          <span className="text-gray-400">{status.playerHandle}</span>
        </div>
      )}

      {status.currentShip && (
        <div className="flex items-center gap-1.5">
          <Ship size={11} className="text-sc-accent2" />
          <span className="text-gray-400">{status.currentShip}</span>
        </div>
      )}

      {status.location && (
        <div className="flex items-center gap-1.5">
          <MapPin size={11} className="text-sc-accent2" />
          <span className="text-gray-400">{status.location}</span>
        </div>
      )}

      <div className="flex-1" />

      {/* Event count */}
      <span className="text-gray-600">
        {status.eventCount || 0} events
      </span>
    </div>
  )
}

export default StatusBar
