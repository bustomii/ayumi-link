import { Plus, X } from 'lucide-react'
import type { ToolTab } from '@/api/toolsTabs'
import { cn } from '@/lib/utils'

type Props = {
  tabs: ToolTab[]
  activeId: string
  onSelect: (id: string) => void
  onClose: (id: string) => void
  onNew: () => void
  busy?: boolean
}

/** Strip tab Terminal/Browser di dalam panel session. */
export function ToolTabBar({ tabs, activeId, onSelect, onClose, onNew, busy }: Props) {
  return (
    <div className="flex shrink-0 items-center gap-1 overflow-x-auto border-b border-border/50 px-2 py-1.5">
      {tabs.map((t) => (
        <div
          key={t.id}
          className={cn(
            'group flex max-w-[10rem] items-center gap-1 rounded-md px-2 py-1 text-xs transition-colors',
            t.id === activeId ? 'bg-accent/20 text-foreground' : 'text-muted-foreground hover:bg-white/5',
          )}
        >
          <button
            type="button"
            className="min-w-0 flex-1 truncate text-left"
            onClick={() => onSelect(t.id)}
            title={t.title || t.id}
          >
            {t.title || 'Tab'}
            {t.alive === false && <span className="ml-1 opacity-50">·</span>}
          </button>
          <button
            type="button"
            className="rounded p-0.5 opacity-60 hover:bg-white/10 hover:opacity-100"
            onClick={(e) => {
              e.stopPropagation()
              onClose(t.id)
            }}
            aria-label="Tutup tab"
          >
            <X className="h-3 w-3" />
          </button>
        </div>
      ))}
      <button
        type="button"
        disabled={busy || tabs.length >= 5}
        onClick={onNew}
        className="flex h-7 w-7 shrink-0 items-center justify-center rounded-md text-muted-foreground hover:bg-white/5 hover:text-foreground disabled:opacity-40"
        title="Tab baru"
        aria-label="Tab baru"
      >
        <Plus className="h-4 w-4" />
      </button>
    </div>
  )
}
