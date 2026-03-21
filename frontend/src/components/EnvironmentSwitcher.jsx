import { useEffect, useRef } from 'react'
import { Server } from 'lucide-react'

function EnvironmentSwitcher({ current, onSelect, onClose }) {
  const ref = useRef(null)

  // Close on click outside or Escape
  useEffect(() => {
    const handleKey = (e) => {
      if (e.key === 'Escape') onClose()
    }
    const handleClick = (e) => {
      if (ref.current && !ref.current.contains(e.target)) onClose()
    }
    window.addEventListener('keydown', handleKey)
    window.addEventListener('mousedown', handleClick)
    return () => {
      window.removeEventListener('keydown', handleKey)
      window.removeEventListener('mousedown', handleClick)
    }
  }, [onClose])

  const envs = [
    { id: 'production', label: 'Production', desc: 'scbridge.app' },
    { id: 'staging', label: 'Staging', desc: 'staging.scbridge.app' },
  ]

  return (
    <div style={{
      position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.5)',
      display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 100,
    }}>
      <div ref={ref} className="bg-sc-dark border border-white/[0.08] rounded-xl" style={{ width: 320, padding: 20 }}>
        <div className="flex items-center gap-2 mb-4">
          <Server size={16} className="text-sc-accent" />
          <h2 className="font-[family-name:var(--font-display)] text-sm tracking-wider text-gray-300 uppercase">
            Environment
          </h2>
        </div>

        <div className="space-y-2">
          {envs.map(env => (
            <button
              key={env.id}
              onClick={() => onSelect(env.id)}
              className="w-full text-left rounded-lg transition-all"
              style={{
                padding: '12px 16px',
                background: current === env.id ? 'rgba(34,211,238,0.08)' : 'rgba(255,255,255,0.02)',
                border: current === env.id ? '1px solid rgba(34,211,238,0.2)' : '1px solid rgba(255,255,255,0.04)',
                cursor: 'pointer',
              }}
            >
              <div className="flex items-center justify-between">
                <span className="text-sm text-gray-200">{env.label}</span>
                {current === env.id && (
                  <div style={{ width: 8, height: 8, borderRadius: '50%', background: '#22d3ee' }} />
                )}
              </div>
              <div className="text-xs text-gray-500 mt-0.5 font-[family-name:var(--font-mono)]">
                {env.desc}
              </div>
            </button>
          ))}
        </div>

        <p className="text-xs text-gray-600 mt-3 text-center">
          Ctrl+Shift+D to toggle
        </p>
      </div>
    </div>
  )
}

export default EnvironmentSwitcher
