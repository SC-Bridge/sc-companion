import { useState, useEffect, useCallback } from 'react'
import { Download } from 'lucide-react'
import StatusBar from './components/StatusBar'
import Dashboard from './components/Dashboard'
import EventFeed from './components/EventFeed'
import Friends from './components/Friends'
import Settings from './components/Settings'
import EnvironmentSwitcher from './components/EnvironmentSwitcher'

// Wails runtime bindings
const wails = window.go?.main?.App

const TABS = [
  { id: 'dashboard', label: 'Dashboard' },
  { id: 'events', label: 'Events' },
  { id: 'friends', label: 'Friends' },
  { id: 'settings', label: 'Settings' },
]

function App() {
  const [status, setStatus] = useState(null)
  const [config, setConfig] = useState(null)
  const [events, setEvents] = useState([])
  const [activeTab, setActiveTab] = useState('dashboard')
  const [showEnvSwitcher, setShowEnvSwitcher] = useState(false)
  const [updateInfo, setUpdateInfo] = useState(null)

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

  // Listen for live events — always active
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
      if (wails) wails.GetConfig().then(setConfig).catch(console.error)
    })
    return () => { if (cancel) cancel() }
  }, [])

  // Check for updates on mount, then every 4 hours
  useEffect(() => {
    if (!wails) return
    const check = async () => {
      try {
        const info = await wails.CheckForUpdate()
        if (info?.hasUpdate) setUpdateInfo(info)
      } catch (e) {
        // Silent — update check is non-critical
      }
    }
    check()
    const interval = setInterval(check, 4 * 60 * 60 * 1000)
    return () => clearInterval(interval)
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

  // Dev mode fallback
  if (!wails) {
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
      {/* Update banner */}
      {updateInfo?.hasUpdate && (
        <UpdateBanner
          info={updateInfo}
          onDismiss={() => setUpdateInfo(null)}
        />
      )}

      {/* Tab nav */}
      <nav className="app-nav" style={{
        display: 'flex',
        alignItems: 'center',
        gap: 2,
        paddingTop: 6,
        paddingBottom: 0,
        borderBottom: '1px solid rgba(255,255,255,0.06)',
      }}>
        {TABS.map(tab => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id)}
            className="font-[family-name:var(--font-display)] tracking-wider uppercase"
            style={{
              position: 'relative',
              padding: '10px 18px',
              fontSize: 13,
              color: activeTab === tab.id ? '#22d3ee' : '#6b7280',
              background: 'none',
              border: 'none',
              cursor: 'pointer',
              transition: 'color 0.2s',
            }}
            onMouseEnter={(e) => {
              if (activeTab !== tab.id) e.currentTarget.style.color = '#9ca3af'
            }}
            onMouseLeave={(e) => {
              if (activeTab !== tab.id) e.currentTarget.style.color = '#6b7280'
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
        {activeTab === 'dashboard' && <Dashboard status={status} config={config} />}
        {activeTab === 'events' && <EventFeed events={events} />}
        {activeTab === 'friends' && <Friends config={config} />}
        {activeTab === 'settings' && <Settings config={config} onConfigChange={setConfig} onUpdateFound={setUpdateInfo} />}
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

function UpdateBanner({ info, onDismiss }) {
  const [updating, setUpdating] = useState(false)
  const [error, setError] = useState(null)

  const handleUpdate = useCallback(async () => {
    const updateUrl = info.installerUrl || info.downloadUrl
    if (!wails || !updateUrl) return
    setUpdating(true)
    setError(null)
    try {
      const err = await wails.ApplyUpdate(updateUrl)
      if (err) {
        setError(err)
        setUpdating(false)
      }
      // If no error, the app is quitting — don't update state
    } catch (e) {
      setError(e.message || 'Update failed')
      setUpdating(false)
    }
  }, [info])

  return (
    <div className="app-nav" style={{
      display: 'flex', alignItems: 'center', gap: 8,
      paddingTop: 6, paddingBottom: 6,
      background: error ? 'rgba(239,68,68,0.06)' : 'rgba(34,211,238,0.06)',
      borderBottom: `1px solid ${error ? 'rgba(239,68,68,0.12)' : 'rgba(34,211,238,0.12)'}`,
      fontSize: 12,
    }}>
      <Download size={13} style={{ color: error ? '#ef4444' : '#22d3ee' }} />
      <span className="text-gray-300">
        {updating ? 'Updating...' : error ? error : `v${info.version} available`}
      </span>
      {!updating && !error && (
        <button
          onClick={handleUpdate}
          className="font-[family-name:var(--font-mono)] cursor-pointer"
          style={{
            padding: '2px 8px', fontSize: 11, borderRadius: 4,
            background: 'rgba(34,211,238,0.15)', border: '1px solid rgba(34,211,238,0.2)',
            color: '#22d3ee', cursor: 'pointer',
          }}
        >
          Update & Restart
        </button>
      )}
      {!updating && (
        <button
          onClick={onDismiss}
          className="cursor-pointer"
          style={{ marginLeft: 'auto', background: 'none', border: 'none', color: '#6b7280', cursor: 'pointer', fontSize: 14 }}
        >
          ×
        </button>
      )}
      {updating && (
        <span className="text-gray-500 font-[family-name:var(--font-mono)]" style={{ marginLeft: 'auto', fontSize: 11 }}>
          Downloading...
        </span>
      )}
    </div>
  )
}

export default App
