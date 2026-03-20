import { Settings as SettingsIcon, FolderOpen, Key, Radio, Globe } from 'lucide-react'

function Settings({ config }) {
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
          Configuration
        </h2>
      </div>

      {/* Data Sources */}
      <SettingsSection title="Data Sources">
        <SettingRow
          icon={FolderOpen}
          label="Game.log Path"
          value={config.logPath || 'Auto-detect'}
          description="Path to Star Citizen Game.log file. Leave empty for auto-detection."
        />
        <SettingRow
          icon={Radio}
          label="gRPC Proxy"
          value={config.proxyEnabled ? `Enabled (port ${config.proxyPort})` : 'Disabled'}
          description="Intercepts game-to-CIG backend traffic for wallet, reputation, and blueprint data."
        />
      </SettingsSection>

      {/* API Connection */}
      <SettingsSection title="SC Bridge API">
        <SettingRow
          icon={Globe}
          label="API Endpoint"
          value={config.apiEndpoint || 'Not set'}
          description="SC Bridge web app endpoint for data sync."
        />
        <SettingRow
          icon={Key}
          label="API Token"
          value={config.apiToken ? '••••••••' : 'Not configured'}
          description="Authentication token for syncing events to scbridge.app."
        />
      </SettingsSection>

      {/* Info */}
      <div className="bg-white/[0.02] border border-white/[0.06] rounded-xl p-4 text-xs text-gray-600">
        <p>
          Configuration is stored in <span className="font-[family-name:var(--font-mono)] text-gray-500">config.yaml</span>.
          Changes require restarting the companion app.
        </p>
        <p className="mt-2">
          CA certificate for gRPC interception is stored in the app data directory.
          On Windows: <span className="font-[family-name:var(--font-mono)] text-gray-500">%APPDATA%\SCBridge\ca.crt</span>
        </p>
      </div>
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
