import { Wifi, WifiOff, Radio, User, Ship, MapPin } from 'lucide-react'

function StatusBar({ status }) {
  if (!status) return null

  return (
    <div className="flex items-center h-7 px-8 bg-sc-darker/60 border-t border-white/[0.04] text-[11px] font-[family-name:var(--font-mono)] text-gray-500 gap-4 select-none">
      {/* Proxy status */}
      <div className="flex items-center gap-1.5">
        {status.proxyRunning ? (
          <>
            <Radio size={11} className="text-sc-accent" />
            <span className="text-sc-accent/80">Proxy</span>
          </>
        ) : (
          <>
            <WifiOff size={11} />
            <span>Proxy Off</span>
          </>
        )}
      </div>

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
