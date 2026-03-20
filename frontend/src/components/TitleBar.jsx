import { Minus, Square, X } from 'lucide-react'

function TitleBar({ isDev }) {
  const minimize = () => window.runtime?.WindowMinimise()
  const toggleMax = () => window.runtime?.WindowToggleMaximise()
  const close = () => window.runtime?.Quit()

  return (
    <div className="titlebar flex items-center h-9 bg-sc-darker/80 border-b border-white/[0.04] select-none">
      {/* Title only — no logo in titlebar */}
      <div className="flex items-center px-4">
        <span className="font-[family-name:var(--font-display)] text-xs tracking-[0.2em] text-gray-400 uppercase">
          SC Bridge Companion
        </span>
      </div>

      <div className="flex-1" />

      {/* Window controls */}
      {!isDev && (
        <div className="flex h-full">
          <button
            onClick={minimize}
            className="w-11 h-full flex items-center justify-center text-gray-500 hover:text-gray-300 hover:bg-white/[0.04] transition-colors"
          >
            <Minus size={14} />
          </button>
          <button
            onClick={toggleMax}
            className="w-11 h-full flex items-center justify-center text-gray-500 hover:text-gray-300 hover:bg-white/[0.04] transition-colors"
          >
            <Square size={11} />
          </button>
          <button
            onClick={close}
            className="w-11 h-full flex items-center justify-center text-gray-500 hover:text-white hover:bg-red-500/80 transition-colors"
          >
            <X size={14} />
          </button>
        </div>
      )}
    </div>
  )
}

export default TitleBar
