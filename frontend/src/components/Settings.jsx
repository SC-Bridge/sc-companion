import { useState, useEffect, useCallback } from 'react'
import { Settings as SettingsIcon, FolderOpen, Globe, Link, Unlink, RotateCcw, ChevronDown, ChevronRight, RefreshCw, Download, Database, FileText } from 'lucide-react'

const wails = window.go?.main?.App

function Settings({ config, onConfigChange, onUpdateFound }) {
  const [categories, setCategories] = useState([])
  const [syncPrefs, setSyncPrefs] = useState({})
  const [expandedCategories, setExpandedCategories] = useState({})
  const [connecting, setConnecting] = useState(false)
  const [version, setVersion] = useState('')
  const [checkingUpdate, setCheckingUpdate] = useState(false)
  const [updateResult, setUpdateResult] = useState(null)
  const [eventLogPath, setEventLogPath] = useState('')
  const [dbPath, setDbPath] = useState('')

  useEffect(() => {
    if (!wails) return
    wails.GetEventCategories().then(setCategories).catch(console.error)
    wails.GetSyncPreferences().then(setSyncPrefs).catch(console.error)
    wails.GetVersion().then(setVersion).catch(console.error)
    wails.GetEventLogPath().then(setEventLogPath).catch(console.error)
    wails.GetDatabasePath().then(setDbPath).catch(console.error)
  }, [])

  const checkForUpdate = useCallback(async () => {
    if (!wails) return
    setCheckingUpdate(true)
    setUpdateResult(null)
    try {
      const info = await wails.CheckForUpdate()
      setUpdateResult(info)
      if (info?.hasUpdate && onUpdateFound) {
        onUpdateFound(info)
      }
    } catch (e) {
      setUpdateResult({ error: e.message || 'Check failed' })
    }
    setCheckingUpdate(false)
  }, [onUpdateFound])

  const handleConnect = useCallback(async () => {
    if (!wails) return
    setConnecting(true)
    try {
      await wails.ConnectToSCBridge()
      const cfg = await wails.GetConfig()
      onConfigChange(cfg)
    } catch (e) {
      console.error('Connect failed:', e)
    }
    setConnecting(false)
  }, [onConfigChange])

  const handleDisconnect = useCallback(async () => {
    if (!wails) return
    await wails.DisconnectFromSCBridge()
    const cfg = await wails.GetConfig()
    onConfigChange(cfg)
  }, [onConfigChange])

  const togglePref = useCallback(async (type, enabled) => {
    if (!wails) return
    await wails.SetSyncPreference(type, enabled)
    setSyncPrefs(prev => ({ ...prev, [type]: enabled }))
  }, [])

  const resetPrefs = useCallback(async () => {
    if (!wails) return
    const defaults = await wails.ResetSyncPreferences()
    setSyncPrefs(defaults)
  }, [])

  const toggleCategory = useCallback((name) => {
    setExpandedCategories(prev => ({ ...prev, [name]: !prev[name] }))
  }, [])

  if (!config) {
    return (
      <div style={{ maxWidth: 720, margin: '0 auto', display: 'flex', alignItems: 'center', justifyContent: 'center', height: 256, color: '#4b5563' }}>
        Loading settings...
      </div>
    )
  }

  return (
    <div style={{ maxWidth: 720, margin: '0 auto', display: 'flex', flexDirection: 'column', gap: 20 }}>
      {/* Header */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 4 }}>
        <SettingsIcon size={16} style={{ color: '#22d3ee' }} />
        <h2 className="font-[family-name:var(--font-display)]" style={{ fontSize: 14, letterSpacing: '0.08em', color: '#9ca3af', textTransform: 'uppercase' }}>
          Settings
        </h2>
      </div>

      {/* SC Bridge Connection */}
      <Section title="SC Bridge Connection">
        {config.connected ? (
          <div style={{ padding: '14px 16px', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
              <Link size={20} style={{ color: '#22d3ee', flexShrink: 0 }} />
              <div>
                <div style={{ fontSize: 14, color: '#e5e7eb' }}>Connected</div>
                <div className="font-[family-name:var(--font-mono)]" style={{ fontSize: 12, color: '#6b7280', marginTop: 1 }}>
                  {config.handle || config.apiEndpoint}
                </div>
              </div>
            </div>
            <button
              onClick={handleDisconnect}
              style={{
                display: 'flex', alignItems: 'center', gap: 6,
                padding: '6px 12px', fontSize: 12, borderRadius: 6,
                background: 'rgba(239,68,68,0.1)', border: '1px solid rgba(239,68,68,0.2)',
                color: '#ef4444', cursor: 'pointer',
              }}
            >
              <Unlink size={12} />
              Disconnect
            </button>
          </div>
        ) : (
          <div style={{ padding: '14px 16px', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
              <Globe size={20} style={{ color: '#6b7280', flexShrink: 0 }} />
              <div>
                <div style={{ fontSize: 14, color: '#9ca3af' }}>Not connected</div>
                <div style={{ fontSize: 12, color: '#4b5563', marginTop: 1 }}>
                  Connect to sync events to scbridge.app
                </div>
              </div>
            </div>
            <button
              onClick={handleConnect}
              disabled={connecting}
              style={{
                display: 'flex', alignItems: 'center', gap: 6,
                padding: '6px 12px', fontSize: 12, borderRadius: 6,
                background: connecting ? 'rgba(34,211,238,0.05)' : 'rgba(34,211,238,0.1)',
                border: '1px solid rgba(34,211,238,0.2)',
                color: '#22d3ee',
                cursor: connecting ? 'wait' : 'pointer',
                opacity: connecting ? 0.6 : 1,
              }}
            >
              <Link size={12} />
              {connecting ? 'Connecting...' : 'Connect'}
            </button>
          </div>
        )}
      </Section>

      {/* Event Sync */}
      <Section title="Event Sync">
        <div style={{ padding: '8px 16px', display: 'flex', alignItems: 'center', justifyContent: 'space-between', borderBottom: '1px solid rgba(255,255,255,0.04)' }}>
          <span style={{ fontSize: 12, color: '#6b7280' }}>Choose which events sync to SC Bridge</span>
          <button
            onClick={resetPrefs}
            style={{ display: 'flex', alignItems: 'center', gap: 4, fontSize: 12, color: '#6b7280', background: 'none', border: 'none', cursor: 'pointer', padding: '2px 6px' }}
          >
            <RotateCcw size={11} />
            Reset
          </button>
        </div>
        {categories.map(cat => {
          const expanded = expandedCategories[cat.name] ?? false
          const enabledCount = cat.events.filter(e => syncPrefs[e.type]).length
          return (
            <div key={cat.name}>
              <button
                onClick={() => toggleCategory(cat.name)}
                style={{
                  width: '100%', display: 'flex', alignItems: 'center', gap: 8,
                  padding: '10px 16px', background: 'none', border: 'none',
                  cursor: 'pointer', textAlign: 'left',
                }}
              >
                {expanded
                  ? <ChevronDown size={14} style={{ color: '#6b7280' }} />
                  : <ChevronRight size={14} style={{ color: '#6b7280' }} />
                }
                <span style={{ flex: 1, fontSize: 14, color: '#d1d5db' }}>{cat.name}</span>
                <span className="font-[family-name:var(--font-mono)]" style={{ fontSize: 12, color: '#4b5563' }}>
                  {enabledCount}/{cat.events.length}
                </span>
              </button>
              {expanded && (
                <div style={{ paddingLeft: 40, paddingRight: 16, paddingBottom: 8 }}>
                  {cat.events.map(evt => (
                    <label
                      key={evt.type}
                      style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '4px 0', cursor: 'pointer' }}
                    >
                      <input
                        type="checkbox"
                        checked={!!syncPrefs[evt.type]}
                        onChange={(e) => togglePref(evt.type, e.target.checked)}
                        style={{ width: 14, height: 14, accentColor: '#22d3ee' }}
                      />
                      <span style={{ fontSize: 12, color: '#9ca3af' }}>{evt.label}</span>
                      <span className="font-[family-name:var(--font-mono)]" style={{ fontSize: 12, color: '#374151' }}>{evt.type}</span>
                    </label>
                  ))}
                </div>
              )}
            </div>
          )
        })}
      </Section>

      {/* Data Sources */}
      <Section title="Data & Files">
        <div style={{ padding: '12px 16px', display: 'flex', alignItems: 'center', gap: 12, borderBottom: '1px solid rgba(255,255,255,0.04)' }}>
          <FolderOpen size={18} style={{ color: '#5b9bd5', flexShrink: 0 }} />
          <div style={{ flex: 1, minWidth: 0 }}>
            <div style={{ fontSize: 13, color: '#d1d5db' }}>Game.log</div>
            <div className="font-[family-name:var(--font-mono)]" style={{
              fontSize: 11, color: '#6b7280', marginTop: 2,
              overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
            }}>
              {config.logPath || 'Auto-detect'}
            </div>
          </div>
          <button
            onClick={async () => {
              if (!wails) return
              const path = await wails.BrowseGameLog()
              if (path) {
                const cfg = await wails.GetConfig()
                onConfigChange(cfg)
              }
            }}
            style={{
              padding: '5px 10px', fontSize: 11, borderRadius: 6,
              background: 'rgba(255,255,255,0.03)', border: '1px solid rgba(255,255,255,0.06)',
              color: '#9ca3af', cursor: 'pointer', flexShrink: 0,
            }}
          >
            Browse
          </button>
          {config.logPath && (
            <button
              onClick={() => wails?.OpenInExplorer(config.logPath)}
              style={{
                padding: '5px 10px', fontSize: 11, borderRadius: 6,
                background: 'rgba(255,255,255,0.03)', border: '1px solid rgba(255,255,255,0.06)',
                color: '#9ca3af', cursor: 'pointer', flexShrink: 0,
              }}
            >
              Reveal
            </button>
          )}
        </div>
        <FileRow
          icon={FileText}
          label="Event Log"
          description="JSONL event log (for WingmanAI)"
          path={eventLogPath}
          onReveal={() => wails?.OpenInExplorer(eventLogPath)}
        />
        <FileRow
          icon={Database}
          label="Database"
          description="Local SQLite event store"
          path={dbPath}
          onReveal={() => wails?.OpenInExplorer(dbPath)}
        />
      </Section>

      {/* About */}
      <Section title="About">
        <div style={{ padding: '14px 16px' }}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
              <Download size={20} style={{ color: '#5b9bd5', flexShrink: 0 }} />
              <div>
                <div style={{ fontSize: 14, color: '#d1d5db' }}>SC Bridge Companion</div>
                <div className="font-[family-name:var(--font-mono)]" style={{ fontSize: 12, color: '#4b5563', marginTop: 1 }}>
                  v{version || '...'}
                </div>
              </div>
            </div>
            <button
              onClick={checkForUpdate}
              disabled={checkingUpdate}
              style={{
                display: 'flex', alignItems: 'center', gap: 6,
                padding: '6px 12px', fontSize: 12, borderRadius: 6,
                background: 'rgba(255,255,255,0.03)', border: '1px solid rgba(255,255,255,0.06)',
                color: checkingUpdate ? '#4b5563' : '#9ca3af',
                cursor: checkingUpdate ? 'wait' : 'pointer',
              }}
            >
              <RefreshCw size={12} className={checkingUpdate ? 'animate-spin' : ''} />
              {checkingUpdate ? 'Checking...' : 'Check for Updates'}
            </button>
          </div>
          {updateResult && (
            <div className="font-[family-name:var(--font-mono)]" style={{
              marginTop: 10, padding: '6px 10px', fontSize: 12, borderRadius: 6,
              background: updateResult.hasUpdate ? 'rgba(34,211,238,0.06)' : 'rgba(255,255,255,0.02)',
              border: updateResult.hasUpdate ? '1px solid rgba(34,211,238,0.12)' : '1px solid rgba(255,255,255,0.04)',
            }}>
              {updateResult.error
                ? <span style={{ color: '#ef4444' }}>{updateResult.error}</span>
                : updateResult.hasUpdate
                  ? <span style={{ color: '#22d3ee' }}>v{updateResult.version} available — use the update banner to install</span>
                  : <span style={{ color: '#6b7280' }}>You're on the latest version</span>
              }
            </div>
          )}
        </div>
      </Section>
    </div>
  )
}

function Section({ title, children }) {
  return (
    <div style={{
      background: 'rgba(255,255,255,0.03)',
      border: '1px solid rgba(255,255,255,0.06)',
      borderRadius: 12,
      overflow: 'hidden',
    }}>
      <div style={{ padding: '10px 16px', borderBottom: '1px solid rgba(255,255,255,0.06)' }}>
        <h3 className="font-[family-name:var(--font-display)]" style={{ fontSize: 11, letterSpacing: '0.06em', color: '#6b7280', textTransform: 'uppercase' }}>
          {title}
        </h3>
      </div>
      {children}
    </div>
  )
}

function FileRow({ icon: Icon, label, description, path, onReveal }) {
  return (
    <div
      onClick={onReveal || undefined}
      style={{
        padding: '12px 16px', display: 'flex', alignItems: 'center', gap: 12,
        borderBottom: '1px solid rgba(255,255,255,0.04)',
        cursor: onReveal ? 'pointer' : 'default',
        transition: 'background 0.15s',
      }}
      onMouseEnter={(e) => { if (onReveal) e.currentTarget.style.background = 'rgba(255,255,255,0.02)' }}
      onMouseLeave={(e) => { e.currentTarget.style.background = 'transparent' }}
    >
      <Icon size={18} style={{ color: '#5b9bd5', flexShrink: 0 }} />
      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{ fontSize: 13, color: '#d1d5db' }}>{label}</div>
        <div style={{ fontSize: 11, color: '#4b5563', marginTop: 1 }}>{description}</div>
      </div>
      <span className="font-[family-name:var(--font-mono)]" style={{
        fontSize: 11, color: '#6b7280', flexShrink: 0, maxWidth: 280,
        overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
      }}>
        {path || '—'}
      </span>
    </div>
  )
}

export default Settings
