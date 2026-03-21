import { useState, useEffect, useCallback } from 'react'
import StatusBar from './components/StatusBar'
import Dashboard from './components/Dashboard'
import EventFeed from './components/EventFeed'
import Settings from './components/Settings'
import EnvironmentSwitcher from './components/EnvironmentSwitcher'

// Wails runtime bindings
const wails = window.go?.main?.App

function App() {
  const [status, setStatus] = useState(null)
  const [config, setConfig] = useState(null)
  const [events, setEvents] = useState([])
  const [activeTab, setActiveTab] = useState('dashboard')
  const [showEnvSwitcher, setShowEnvSwitcher] = useState(false)

  // Poll status every 2 seconds
  useEffect(() => {
    const fetchStatus = async () => {
      if (!wails) return
      try {
        const s = await wails.GetStatus()
        setStatus(s)
      } catch (e) {
        console.error('GetStatus failed:', e)
      }
    }
    fetchStatus()
    const interval = setInterval(fetchStatus, 2000)
    return () => clearInterval(interval)
  }, [])

  // Load config once
  useEffect(() => {
    if (!wails) return
    wails.GetConfig().then(setConfig).catch(console.error)
  }, [])

  // Listen for live events — always active (no debug gate)
  useEffect(() => {
    if (!window.runtime) return
    const cancel = window.runtime.EventsOn('event', (entry) => {
      setEvents(prev => {
        const next = [...prev, entry]
        return next.length > 200 ? next.slice(-200) : next
      })
    })
    return () => { if (cancel) cancel() }
  }, [])

  // Load buffered events on mount
  useEffect(() => {
    if (!wails) return
    wails.GetRecentEvents().then(setEvents).catch(console.error)
  }, [])

  // Listen for auth expiry
  useEffect(() => {
    if (!window.runtime) return
    const cancel = window.runtime.EventsOn('auth_expired', () => {
      // Refresh config to update connection state
      if (wails) wails.GetConfig().then(setConfig).catch(console.error)
    })
    return () => { if (cancel) cancel() }
  }, [])

  // Ctrl+Shift+D toggles environment switcher
  useEffect(() => {
    const handleKeyDown = (e) => {
      if (e.ctrlKey && e.shiftKey && e.key === 'D') {
        e.preventDefault()
        setShowEnvSwitcher(prev => !prev)
      }
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [])

  const handleEnvChange = useCallback(async (env) => {
    if (!wails) return
    await wails.SetEnvironment(env)
    const cfg = await wails.GetConfig()
    setConfig(cfg)
    setShowEnvSwitcher(false)
  }, [])

  // Dev mode fallback when not running in Wails
  const isDev = !wails
  if (isDev) {
    return (
      <div style={{ display: 'flex', flexDirection: 'column', width: '100%', height: '100%' }}>
        <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <div style={{ textAlign: 'center' }}>
            <h2 className="font-[family-name:var(--font-display)] text-xl text-white tracking-wider" style={{ marginBottom: 8 }}>
              SC BRIDGE COMPANION
            </h2>
            <p className="text-gray-500 text-sm">
              Run with <span className="font-[family-name:var(--font-mono)] text-sc-accent/60">wails dev</span> to connect to the Go backend
            </p>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', width: '100%', height: '100%', overflow: 'hidden' }}>
      {/* Tab nav */}
      <nav className="app-nav" style={{ display: 'flex', alignItems: 'center', gap: 4, paddingTop: 8, borderBottom: '1px solid rgba(255,255,255,0.06)' }}>
        {[
          { id: 'dashboard', label: 'Dashboard' },
          { id: 'events', label: 'Event Feed' },
          { id: 'settings', label: 'Settings' },
        ].map(tab => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id)}
            className="font-[family-name:var(--font-display)] tracking-wider uppercase"
            style={{
              position: 'relative',
              padding: '10px 20px',
              fontSize: 14,
              color: activeTab === tab.id ? '#22d3ee' : '#6b7280',
              background: 'none',
              border: 'none',
              cursor: 'pointer',
              transition: 'color 0.2s',
            }}
          >
            {tab.label}
            {activeTab === tab.id && (
              <div style={{
                position: 'absolute',
                bottom: 0,
                left: 8,
                right: 8,
                height: 2,
                background: '#22d3ee',
                borderRadius: 1,
                boxShadow: '0 0 8px rgba(34,211,238,0.4)',
              }} />
            )}
          </button>
        ))}
      </nav>

      {/* Content */}
      <main className="app-content" style={{ flex: 1, overflowY: 'auto' }}>
        {activeTab === 'dashboard' && <Dashboard status={status} />}
        {activeTab === 'events' && <EventFeed events={events} />}
        {activeTab === 'settings' && <Settings config={config} onConfigChange={setConfig} />}
      </main>

      <StatusBar status={status} />

      {showEnvSwitcher && (
        <EnvironmentSwitcher
          current={config?.environment || 'production'}
          onSelect={handleEnvChange}
          onClose={() => setShowEnvSwitcher(false)}
        />
      )}
    </div>
  )
}

export default App
