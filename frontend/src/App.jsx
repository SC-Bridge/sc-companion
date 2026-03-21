import { useState, useEffect, useCallback } from 'react'
import StatusBar from './components/StatusBar'
import Dashboard from './components/Dashboard'
import EventFeed from './components/EventFeed'
import GrpcExplorer from './components/GrpcExplorer'
import Settings from './components/Settings'

// Wails runtime bindings
const wails = window.go?.main?.App

function App() {
  const [status, setStatus] = useState(null)
  const [config, setConfig] = useState(null)
  const [events, setEvents] = useState([])
  const [activeTab, setActiveTab] = useState('dashboard')
  const [debugMode, setDebugMode] = useState(false)

  // Poll status every 2 seconds
  useEffect(() => {
    const fetchStatus = async () => {
      if (!wails) return
      try {
        const s = await wails.GetStatus()
        setStatus(s)
        setDebugMode(s.debugMode)
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

  // Listen for live events (debug mode)
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

  // Load buffered events when switching to debug
  useEffect(() => {
    if (debugMode && wails) {
      wails.GetRecentEvents().then(setEvents).catch(console.error)
    }
  }, [debugMode])

  const toggleDebug = useCallback(async () => {
    if (!wails) return
    const next = !debugMode
    await wails.SetDebugMode(next)
    setDebugMode(next)
    if (next) {
      setActiveTab('events')
    }
  }, [debugMode])

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
          { id: 'explorer', label: 'gRPC Explorer' },
          ...(debugMode ? [{ id: 'events', label: 'Event Feed' }] : []),
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
        <div style={{ flex: 1 }} />
        <button
          onClick={toggleDebug}
          className="font-[family-name:var(--font-mono)]"
          style={{
            padding: '6px 12px',
            fontSize: 12,
            borderRadius: 6,
            border: debugMode ? '1px solid rgba(34,211,238,0.2)' : '1px solid rgba(255,255,255,0.06)',
            background: debugMode ? 'rgba(34,211,238,0.1)' : 'transparent',
            color: debugMode ? '#22d3ee' : '#4b5563',
            cursor: 'pointer',
            transition: 'all 0.2s',
          }}
        >
          {debugMode ? 'DEBUG ON' : 'DEBUG'}
        </button>
      </nav>

      {/* Content */}
      <main className="app-content" style={{ flex: 1, overflowY: 'auto' }}>
        {activeTab === 'dashboard' && <Dashboard status={status} />}
        {activeTab === 'explorer' && <GrpcExplorer />}
        {activeTab === 'events' && <EventFeed events={events} />}
        {activeTab === 'settings' && <Settings config={config} />}
      </main>

      <StatusBar status={status} />
    </div>
  )
}

export default App
