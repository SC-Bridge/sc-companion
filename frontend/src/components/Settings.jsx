import { useState, useEffect, useCallback } from 'react'
import { Settings as SettingsIcon, FolderOpen, Globe, Link, Unlink, RotateCcw, ChevronDown, ChevronRight } from 'lucide-react'

const wails = window.go?.main?.App

function Settings({ config, onConfigChange }) {
  const [categories, setCategories] = useState([])
  const [syncPrefs, setSyncPrefs] = useState({})
  const [expandedCategories, setExpandedCategories] = useState({})
  const [connecting, setConnecting] = useState(false)

  // Load event categories and sync preferences
  useEffect(() => {
    if (!wails) return
    wails.GetEventCategories().then(setCategories).catch(console.error)
    wails.GetSyncPreferences().then(setSyncPrefs).catch(console.error)
  }, [])

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
      <div className="max-w-2xl mx-auto flex items-center justify-center h-64 text-gray-600">
        Loading settings...
      </div>
    )
  }

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      <div className="flex items-center gap-2 mb-4">
        <SettingsIcon size={16} className="text-sc-accent" />
        <h2 className="font-[family-name:var(--font-display)] text-sm tracking-wider text-gray-400 uppercase">
          Settings
        </h2>
      </div>

      {/* Section 1: SC Bridge Connection */}
      <SettingsSection title="SC Bridge Connection">
        {config.connected ? (
          <div className="px-4 py-3">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className="w-8 h-8 rounded-lg bg-sc-accent/10 flex items-center justify-center">
                  <Link size={14} className="text-sc-accent" />
                </div>
                <div>
                  <div className="text-sm text-gray-200">Connected</div>
                  <div className="text-xs text-gray-500 font-[family-name:var(--font-mono)]">
                    {config.handle || config.apiEndpoint}
                  </div>
                </div>
              </div>
              <button
                onClick={handleDisconnect}
                className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs transition-all"
                style={{
                  background: 'rgba(239,68,68,0.1)',
                  border: '1px solid rgba(239,68,68,0.2)',
                  color: '#ef4444',
                  cursor: 'pointer',
                }}
              >
                <Unlink size={12} />
                Disconnect
              </button>
            </div>
          </div>
        ) : (
          <div className="px-4 py-3">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className="w-8 h-8 rounded-lg bg-white/[0.03] flex items-center justify-center">
                  <Globe size={14} className="text-gray-500" />
                </div>
                <div>
                  <div className="text-sm text-gray-400">Not connected</div>
                  <div className="text-xs text-gray-600">
                    Connect to sync events to scbridge.app
                  </div>
                </div>
              </div>
              <button
                onClick={handleConnect}
                disabled={connecting}
                className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition-all"
                style={{
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
          </div>
        )}
      </SettingsSection>

      {/* Section 2: Event Sync Toggles */}
      <SettingsSection title="Event Sync">
        <div className="px-4 py-2 border-b border-white/[0.04] flex items-center justify-between">
          <span className="text-xs text-gray-500">Choose which events sync to SC Bridge</span>
          <button
            onClick={resetPrefs}
            className="flex items-center gap-1 text-xs text-gray-500 hover:text-gray-300 transition-colors"
            style={{ background: 'none', border: 'none', cursor: 'pointer', padding: '2px 6px' }}
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
                className="w-full flex items-center gap-2 px-4 py-2.5 hover:bg-white/[0.02] transition-colors"
                style={{ background: 'none', border: 'none', cursor: 'pointer', textAlign: 'left' }}
              >
                {expanded
                  ? <ChevronDown size={14} className="text-gray-500" />
                  : <ChevronRight size={14} className="text-gray-500" />
                }
                <span className="text-sm text-gray-300 flex-1">{cat.name}</span>
                <span className="text-xs text-gray-600 font-[family-name:var(--font-mono)]">
                  {enabledCount}/{cat.events.length}
                </span>
              </button>
              {expanded && (
                <div className="pl-10 pr-4 pb-2 space-y-1">
                  {cat.events.map(evt => (
                    <label
                      key={evt.type}
                      className="flex items-center gap-2.5 py-1 cursor-pointer group"
                    >
                      <input
                        type="checkbox"
                        checked={!!syncPrefs[evt.type]}
                        onChange={(e) => togglePref(evt.type, e.target.checked)}
                        className="accent-[#22d3ee]"
                        style={{ width: 14, height: 14 }}
                      />
                      <span className="text-xs text-gray-400 group-hover:text-gray-300 transition-colors">
                        {evt.label}
                      </span>
                      <span className="text-xs text-gray-700 font-[family-name:var(--font-mono)]">
                        {evt.type}
                      </span>
                    </label>
                  ))}
                </div>
              )}
            </div>
          )
        })}
      </SettingsSection>

      {/* Section 3: Data Sources */}
      <SettingsSection title="Data Sources">
        <SettingRow
          icon={FolderOpen}
          label="Game.log Path"
          value={config.logPath || 'Auto-detect'}
          description="Path to Star Citizen Game.log file."
        />
      </SettingsSection>
    </div>
  )
}

const SettingsSection = ({ title, children }) => (
  <div className="bg-white/[0.03] backdrop-blur-md border border-white/[0.06] rounded-xl overflow-hidden">
    <div className="px-4 py-2.5 border-b border-white/[0.06]">
      <h3 className="font-[family-name:var(--font-display)] text-xs tracking-wider text-gray-500 uppercase">
        {title}
      </h3>
    </div>
    <div className="divide-y divide-white/[0.04]">
      {children}
    </div>
  </div>
)

const SettingRow = ({ icon: Icon, label, value, description }) => (
  <div className="flex items-start gap-3 px-4 py-3">
    <div className="w-8 h-8 rounded-lg bg-white/[0.03] flex items-center justify-center shrink-0 mt-0.5">
      <Icon size={14} className="text-sc-accent2" />
    </div>
    <div className="flex-1 min-w-0">
      <div className="flex items-baseline justify-between gap-2">
        <span className="text-sm text-gray-300">{label}</span>
        <span className="text-xs font-[family-name:var(--font-mono)] text-gray-500 truncate">
          {value}
        </span>
      </div>
      <div className="text-xs text-gray-600 mt-0.5">{description}</div>
    </div>
  </div>
)

export default Settings
