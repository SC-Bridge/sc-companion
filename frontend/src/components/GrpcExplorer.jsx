import { useState, useEffect, useRef, useCallback } from 'react'
import { Search, Play, Copy, Check, ChevronRight, ChevronDown } from 'lucide-react'

const wails = window.go?.main?.App

function GrpcExplorer() {
  const [methods, setMethods] = useState([])
  const [selected, setSelected] = useState('')
  const [pageSize, setPageSize] = useState(100)
  const [result, setResult] = useState(null)
  const [error, setError] = useState(null)
  const [loading, setLoading] = useState(false)
  const [filter, setFilter] = useState('')
  const [copied, setCopied] = useState(false)
  const [collapsed, setCollapsed] = useState({})
  const [splitPos, setSplitPos] = useState(320)
  const containerRef = useRef(null)
  const dragging = useRef(false)

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
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    }
  }

  const handleKeyDown = (e) => {
    if (e.key === 'Enter' && selected) handleCall()
  }

  // Drag to resize
  const onMouseDown = useCallback((e) => {
    e.preventDefault()
    dragging.current = true
    const onMouseMove = (e) => {
      if (!dragging.current || !containerRef.current) return
      const rect = containerRef.current.getBoundingClientRect()
      const pos = Math.max(200, Math.min(e.clientX - rect.left, rect.width - 200))
      setSplitPos(pos)
    }
    const onMouseUp = () => {
      dragging.current = false
      document.removeEventListener('mousemove', onMouseMove)
      document.removeEventListener('mouseup', onMouseUp)
    }
    document.addEventListener('mousemove', onMouseMove)
    document.addEventListener('mouseup', onMouseUp)
  }, [])

  const toggleService = (svc) => {
    setCollapsed(prev => ({ ...prev, [svc]: !prev[svc] }))
  }

  const filtered = filter
    ? methods.filter(m => m.toLowerCase().includes(filter.toLowerCase()))
    : methods

  // Group methods by service — extract a short service name
  const grouped = {}
  for (const m of filtered) {
    const parts = m.split('/')
    const svc = parts.length >= 2 ? parts.slice(0, -1).join('/') : 'unknown'
    const method = parts[parts.length - 1]
    if (!grouped[svc]) grouped[svc] = []
    grouped[svc].push({ full: m, name: method })
  }

  // Extract a short display name from the full service path
  const shortServiceName = (svc) => {
    // e.g. /sc.external.services.reputation.v1.ReputationService → ReputationService
    const last = svc.split('.').pop()
    return last || svc
  }

  const resultText = result
    ? (typeof result === 'string' ? result : JSON.stringify(result, null, 2))
    : null

  return (
    <div className="flex flex-col h-full gap-3" onKeyDown={handleKeyDown}>
      {/* Toolbar */}
      <div className="flex items-center gap-2">
        <div className="relative flex-1">
          <Search size={12} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-gray-600" />
          <input
            type="text"
            placeholder="Filter methods..."
            value={filter}
            onChange={e => setFilter(e.target.value)}
            className="w-full pl-7 pr-3 py-2 text-xs font-[family-name:var(--font-mono)] bg-sc-panel/60 border border-sc-border/30 rounded-lg text-gray-200 placeholder-gray-600 focus:outline-none focus:border-sc-accent/40"
          />
        </div>
        <div className="flex items-center gap-1 bg-sc-panel/60 border border-sc-border/30 rounded-lg px-2">
          <span className="text-[10px] text-gray-500 uppercase tracking-wider">Page</span>
          <input
            type="number"
            value={pageSize}
            onChange={e => setPageSize(Number(e.target.value))}
            className="w-14 px-1 py-2 text-xs font-[family-name:var(--font-mono)] bg-transparent text-gray-200 focus:outline-none text-center"
            min={0}
            max={1000}
          />
        </div>
        <button
          onClick={handleCall}
          disabled={loading || !selected}
          className="flex items-center gap-1.5 px-5 py-2 text-xs font-[family-name:var(--font-display)] tracking-wider uppercase rounded-lg transition-all disabled:opacity-30"
          style={{
            background: loading ? 'rgba(34,211,238,0.05)' : 'rgba(34,211,238,0.15)',
            border: '1px solid rgba(34,211,238,0.3)',
            color: '#22d3ee',
            boxShadow: loading ? 'none' : '0 0 12px rgba(34,211,238,0.1)',
          }}
        >
          <Play size={12} fill={loading ? 'none' : 'currentColor'} />
          {loading ? 'Calling...' : 'Call'}
        </button>
      </div>

      {/* Split pane */}
      <div ref={containerRef} className="flex flex-1 min-h-0 gap-0">
        {/* Method list */}
        <div
          className="shrink-0 overflow-y-auto bg-sc-darker/50 border border-sc-border/20 rounded-l-xl"
          style={{ width: splitPos }}
        >
          {Object.entries(grouped).map(([svc, methods]) => {
            const isCollapsed = collapsed[svc]
            const hasSelected = methods.some(m => m.full === selected)
            return (
              <div key={svc}>
                <button
                  onClick={() => toggleService(svc)}
                  className="w-full flex items-center gap-1.5 px-3 py-2 text-left border-b border-sc-border/10 hover:bg-white/[0.03] transition-colors sticky top-0 z-10"
                  style={{ background: 'rgba(9,19,31,0.95)' }}
                >
                  {isCollapsed
                    ? <ChevronRight size={10} className="text-gray-600 shrink-0" />
                    : <ChevronDown size={10} className="text-gray-600 shrink-0" />
                  }
                  <span className={`text-[11px] font-[family-name:var(--font-display)] tracking-wide truncate ${
                    hasSelected ? 'text-sc-accent/80' : 'text-gray-400'
                  }`}>
                    {shortServiceName(svc)}
                  </span>
                  <span className="text-[9px] text-gray-600 ml-auto shrink-0">{methods.length}</span>
                </button>
                {!isCollapsed && methods.map(({ full, name }) => (
                  <button
                    key={full}
                    onClick={() => setSelected(full)}
                    className={`w-full text-left py-1.5 text-[12px] font-[family-name:var(--font-mono)] border-b border-sc-border/5 transition-colors ${
                      selected === full
                        ? 'bg-sc-accent/10 text-sc-accent'
                        : 'text-gray-300 hover:bg-white/[0.04] hover:text-white'
                    }`}
                    style={{ paddingLeft: 28 }}
                  >
                    {name}
                  </button>
                ))}
              </div>
            )
          })}
        </div>

        {/* Drag handle */}
        <div
          onMouseDown={onMouseDown}
          className="w-1.5 shrink-0 cursor-col-resize hover:bg-sc-accent/20 active:bg-sc-accent/30 transition-colors"
          style={{ background: 'rgba(42,74,107,0.3)' }}
        />

        {/* Result panel */}
        <div className="flex-1 flex flex-col min-w-0 bg-sc-darker/50 border border-sc-border/20 rounded-r-xl overflow-hidden">
          {/* Result header */}
          <div className="flex items-center gap-2 px-4 py-2 border-b border-sc-border/15 bg-sc-darker/30">
            <span className="text-[11px] font-[family-name:var(--font-mono)] text-sc-accent/70 truncate flex-1">
              {selected || 'Select a method'}
            </span>
            {resultText && (
              <button
                onClick={copyResult}
                className="flex items-center gap-1 text-[10px] px-2 py-1 rounded text-gray-500 hover:text-gray-300 hover:bg-white/[0.05] transition-colors"
              >
                {copied ? <Check size={10} className="text-sc-success" /> : <Copy size={10} />}
                {copied ? 'Copied' : 'Copy'}
              </button>
            )}
          </div>

          {/* Result body */}
          <div className="flex-1 overflow-auto p-4">
            {loading && (
              <div className="flex items-center gap-2 text-gray-500 text-xs">
                <div className="w-3 h-3 border border-sc-accent/40 border-t-sc-accent rounded-full animate-spin" />
                Calling {selected.split('/').pop()}...
              </div>
            )}
            {error && (
              <div className="text-xs font-[family-name:var(--font-mono)] leading-relaxed">
                <span className="text-red-400/80">Error: </span>
                <span className="text-red-300">{error}</span>
              </div>
            )}
            {resultText && (
              <pre className="text-[12px] font-[family-name:var(--font-mono)] text-gray-200 whitespace-pre-wrap break-words leading-relaxed">
                {resultText}
              </pre>
            )}
            {!loading && !error && !resultText && (
              <div className="text-gray-500 text-xs leading-relaxed">
                Select a method and click <span className="text-sc-accent/60">Call</span> to see the raw JSON response.
                <br /><br />
                <span className="text-gray-600">
                  Set page size to <span className="text-gray-400">0</span> for methods that don't use pagination
                  (e.g. GetFunds, GetCurrentPlayer).
                </span>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

export default GrpcExplorer
