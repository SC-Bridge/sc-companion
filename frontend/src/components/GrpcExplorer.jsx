import { useState, useEffect } from 'react'
import { Search, Play, Copy, ChevronDown } from 'lucide-react'

const wails = window.go?.main?.App

function GrpcExplorer() {
  const [methods, setMethods] = useState([])
  const [selected, setSelected] = useState('')
  const [pageSize, setPageSize] = useState(100)
  const [result, setResult] = useState(null)
  const [error, setError] = useState(null)
  const [loading, setLoading] = useState(false)
  const [filter, setFilter] = useState('')

  useEffect(() => {
    if (!wails) return
    wails.ListGrpcMethods().then(m => {
      setMethods(m || [])
      if (m && m.length > 0) setSelected(m[0])
    })
  }, [])

  const handleCall = async () => {
    if (!wails || !selected) return
    setLoading(true)
    setResult(null)
    setError(null)
    try {
      const json = await wails.RawGrpcCall(selected, pageSize)
      try {
        setResult(JSON.parse(json))
      } catch {
        setResult(json)
      }
    } catch (e) {
      setError(e.message || String(e))
    }
    setLoading(false)
  }

  const copyResult = () => {
    if (result) {
      navigator.clipboard.writeText(typeof result === 'string' ? result : JSON.stringify(result, null, 2))
    }
  }

  const filtered = filter
    ? methods.filter(m => m.toLowerCase().includes(filter.toLowerCase()))
    : methods

  // Group methods by service
  const grouped = {}
  for (const m of filtered) {
    const parts = m.split('/')
    const svc = parts.length >= 2 ? parts.slice(0, -1).join('/') : 'unknown'
    const method = parts[parts.length - 1]
    if (!grouped[svc]) grouped[svc] = []
    grouped[svc].push({ full: m, name: method })
  }

  return (
    <div className="flex flex-col h-full max-w-5xl mx-auto gap-3">
      {/* Header */}
      <div className="flex items-center gap-2">
        <Search size={14} className="text-sc-accent" />
        <span className="font-[family-name:var(--font-display)] text-xs tracking-wider text-gray-400 uppercase">
          gRPC Explorer
        </span>
        <span className="text-xs text-gray-600">
          {methods.length} methods available
        </span>
      </div>

      {/* Controls */}
      <div className="flex gap-2">
        <div className="relative flex-1">
          <input
            type="text"
            placeholder="Filter methods..."
            value={filter}
            onChange={e => setFilter(e.target.value)}
            className="w-full px-3 py-2 text-xs font-[family-name:var(--font-mono)] bg-white/[0.03] border border-white/[0.06] rounded-lg text-gray-300 placeholder-gray-600 focus:outline-none focus:border-sc-accent/30"
          />
        </div>
        <input
          type="number"
          value={pageSize}
          onChange={e => setPageSize(Number(e.target.value))}
          className="w-20 px-2 py-2 text-xs font-[family-name:var(--font-mono)] bg-white/[0.03] border border-white/[0.06] rounded-lg text-gray-300 focus:outline-none focus:border-sc-accent/30"
          placeholder="Page"
          min={0}
          max={1000}
        />
        <button
          onClick={handleCall}
          disabled={loading || !selected}
          className="flex items-center gap-1.5 px-4 py-2 text-xs font-[family-name:var(--font-display)] tracking-wider uppercase bg-sc-accent/10 border border-sc-accent/20 rounded-lg text-sc-accent hover:bg-sc-accent/20 transition-colors disabled:opacity-40"
        >
          <Play size={12} />
          {loading ? 'Calling...' : 'Call'}
        </button>
      </div>

      <div className="flex gap-3 flex-1 min-h-0">
        {/* Method list */}
        <div className="w-80 shrink-0 overflow-y-auto bg-white/[0.02] border border-white/[0.06] rounded-xl">
          {Object.entries(grouped).map(([svc, methods]) => (
            <div key={svc}>
              <div className="px-3 py-1.5 text-[10px] font-[family-name:var(--font-mono)] text-gray-600 bg-white/[0.02] border-b border-white/[0.03] sticky top-0">
                {svc.replace(/^\//, '')}
              </div>
              {methods.map(({ full, name }) => (
                <button
                  key={full}
                  onClick={() => setSelected(full)}
                  className={`w-full text-left px-3 py-1.5 text-xs font-[family-name:var(--font-mono)] border-b border-white/[0.03] transition-colors ${
                    selected === full
                      ? 'bg-sc-accent/10 text-sc-accent'
                      : 'text-gray-400 hover:bg-white/[0.03] hover:text-gray-300'
                  }`}
                >
                  {name}
                </button>
              ))}
            </div>
          ))}
        </div>

        {/* Result panel */}
        <div className="flex-1 flex flex-col min-w-0">
          {/* Selected method */}
          <div className="flex items-center gap-2 mb-2">
            <span className="text-xs font-[family-name:var(--font-mono)] text-gray-500 truncate">
              {selected || 'Select a method'}
            </span>
            {result && (
              <button onClick={copyResult} className="text-gray-600 hover:text-gray-400 transition-colors" title="Copy JSON">
                <Copy size={12} />
              </button>
            )}
          </div>

          <div className="flex-1 overflow-auto bg-white/[0.02] border border-white/[0.06] rounded-xl p-3">
            {loading && (
              <div className="text-gray-600 text-xs">Calling {selected}...</div>
            )}
            {error && (
              <div className="text-red-400 text-xs font-[family-name:var(--font-mono)] whitespace-pre-wrap">
                Error: {error}
              </div>
            )}
            {result && (
              <pre className="text-xs font-[family-name:var(--font-mono)] text-gray-300 whitespace-pre-wrap break-all leading-relaxed">
                {typeof result === 'string' ? result : JSON.stringify(result, null, 2)}
              </pre>
            )}
            {!loading && !error && !result && (
              <div className="text-gray-600 text-xs">
                Select a method and click Call to see the raw JSON response.
                <br /><br />
                Set page size to 0 for methods that don't use pagination (e.g. GetFunds, GetFriendList).
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

export default GrpcExplorer
