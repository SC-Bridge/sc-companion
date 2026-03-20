import { useState, useEffect, useCallback } from 'react'
import TitleBar from './components/TitleBar'
import StatusBar from './components/StatusBar'
import Dashboard from './components/Dashboard'
import EventFeed from './components/EventFeed'
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
      <div className="flex flex-col h-screen">
        <TitleBar isDev />
        <div className="flex-1 flex items-center justify-center">
          <div className="text-center">
            <div className="w-20 h-20 mx-auto mb-6 rounded-full border-2 border-sc-accent/30 flex items-center justify-center">
              <div className="w-12 h-12 rounded-full border-2 border-sc-accent animate-pulse" />
            </div>
            <h2 className="font-[family-name:var(--font-display)] text-xl text-white tracking-wider mb-2">
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
    <div className="flex flex-col h-screen">
      <TitleBar />
      <div className="flex-1 flex flex-col overflow-hidden">
        {/* Tab nav */}
        <nav className="flex items-center gap-1 px-6 pt-2 border-b border-white/[0.06]">
          {[
            { id: 'dashboard', label: 'Dashboard' },
            ...(debugMode ? [{ id: 'events', label: 'Event Feed' }] : []),
            { id: 'settings', label: 'Settings' },
          ].map(tab => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={`relative px-4 py-2.5 text-sm font-[family-name:var(--font-display)] tracking-wider uppercase transition-colors ${
                activeTab === tab.id
                  ? 'text-sc-accent'
                  : 'text-gray-500 hover:text-gray-300'
              }`}
            >
              {tab.label}
              {activeTab === tab.id && (
                <div className="absolute bottom-0 left-2 right-2 h-0.5 bg-sc-accent rounded-full shadow-[0_0_8px_rgba(34,211,238,0.4)]" />
              )}
            </button>
          ))}
          <div className="flex-1" />
          <button
            onClick={toggleDebug}
            className={`px-3 py-1.5 mr-1 text-xs font-[family-name:var(--font-mono)] rounded transition-colors ${
              debugMode
                ? 'bg-sc-accent/10 text-sc-accent border border-sc-accent/20'
                : 'text-gray-600 hover:text-gray-400 border border-white/[0.06]'
            }`}
          >
            {debugMode ? 'DEBUG ON' : 'DEBUG'}
          </button>
        </nav>

        {/* Content */}
        <main className="flex-1 overflow-y-auto px-6 py-5">
          {activeTab === 'dashboard' && <Dashboard status={status} />}
          {activeTab === 'events' && <EventFeed events={events} />}
          {activeTab === 'settings' && <Settings config={config} />}
        </main>
      </div>
      <StatusBar status={status} />
    </div>
  )
}

export default App
