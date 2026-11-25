import { Check, ChevronLeft, ChevronRight, ChevronUp } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import {
  EFFORTS,
  findModelOption,
  findOption,
  isValidModelId,
  MODELS,
  MORE_MODELS,
  PERMISSION_MODES,
  type Option,
  type Prefs,
} from '@/lib/chat-options'
import { cn } from '@/lib/utils'

type Section = { key: keyof Prefs; title: string; options: Option[] }

const SECTIONS: Section[] = [
  { key: 'model', title: 'Model', options: MODELS },
  { key: 'effort', title: 'Effort', options: EFFORTS },
  { key: 'permissionMode', title: 'Izin tool', options: PERMISSION_MODES },
]

function IconBox({ option }: { option: Option }) {
  const Icon = option.icon
  return (
    <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-background/60">
      <Icon className={cn('h-4 w-4', option.tone)} />
    </span>
  )
}

const rowClass = 'flex w-full items-center gap-2.5 rounded-lg px-2 py-2 text-left transition-colors hover:bg-white/5'

type Props = { prefs: Prefs; disabled: boolean; onChange: (p: Prefs) => void }

/** Satu tombol; menu utama berisi 3 baris (model, effort, izin tool), tiap baris membuka daftar pilihannya. */
export function SettingsMenu({ prefs, disabled, onChange }: Props) {
  const [open, setOpen] = useState(false)
  const [view, setView] = useState<keyof Prefs | null>(null)
  const [showMore, setShowMore] = useState(false)
  const [customModel, setCustomModel] = useState('')
  const boxRef = useRef<HTMLDivElement>(null)

  const model = findModelOption(prefs.model)
  const effort = findOption(EFFORTS, prefs.effort)
  const ModelIcon = model.icon
  const section = SECTIONS.find((s) => s.key === view) ?? null
  const customOk = isValidModelId(customModel.trim())

  useEffect(() => {
    if (!open) return
    const close = () => {
      setOpen(false)
      setView(null)
      setShowMore(false)
    }
    const onDown = (e: MouseEvent) => !boxRef.current?.contains(e.target as Node) && close()
    const onKey = (e: KeyboardEvent) => e.key === 'Escape' && close()
    document.addEventListener('mousedown', onDown)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onDown)
      document.removeEventListener('keydown', onKey)
    }
  }, [open])

  return (
    <div ref={boxRef} className="shrink-0">
      <button
        type="button"
        disabled={disabled}
        onClick={() => setOpen((o) => !o)}
        title="Model, effort, dan izin tool"
        aria-haspopup="menu"
        aria-expanded={open}
        className="flex h-10 items-center gap-1.5 rounded-full border border-border bg-card/70 px-3 text-xs font-medium outline-none transition-colors hover:bg-card focus-visible:border-primary disabled:opacity-60"
      >
        <ModelIcon className={cn('h-4 w-4', model.tone)} />
        <span className="hidden sm:inline">
          {model.label} · {effort.label}
        </span>
        <ChevronUp className={cn('h-3.5 w-3.5 text-muted-foreground transition-transform', !open && 'rotate-180')} />
      </button>

      {open && (
        <div
          role="menu"
          className="absolute right-0 bottom-full z-30 mb-2 max-h-[min(70vh,calc(var(--app-height)-var(--safe-t)-var(--safe-b)-8rem))] w-[min(18rem,100%)] overflow-y-auto rounded-xl border border-border bg-card p-1.5 shadow-xl"
        >
          {!section &&
            SECTIONS.map((sec) => {
              const cur =
                sec.key === 'model' ? findModelOption(prefs.model) : findOption(sec.options, prefs[sec.key])
              return (
                <button key={sec.key} type="button" role="menuitem" onClick={() => setView(sec.key)} className={rowClass}>
                  <IconBox option={cur} />
                  <span className="min-w-0 flex-1">
                    <span className="block text-[11px] text-muted-foreground">{sec.title}</span>
                    <span className="block truncate text-sm font-medium">{cur.label}</span>
                  </span>
                  <ChevronRight className="h-4 w-4 shrink-0 text-muted-foreground" />
                </button>
              )
            })}

          {section && (
            <>
              <button
                type="button"
                onClick={() => setView(null)}
                className="mb-1 flex w-full items-center gap-1.5 rounded-lg px-2 py-1.5 text-left text-[11px] font-medium text-muted-foreground transition-colors hover:bg-white/5"
              >
                <ChevronLeft className="h-3.5 w-3.5" />
                {section.title}
              </button>
              {[...section.options, ...(section.key === 'model' && showMore ? MORE_MODELS : [])].map((o) => {
                const selected = o.value === prefs[section.key]
                return (
                  <button
                    key={o.value}
                    type="button"
                    role="menuitemradio"
                    aria-checked={selected}
                    onClick={() => {
                      onChange({ ...prefs, [section.key]: o.value })
                      setView(null)
                    }}
                    className={cn(rowClass, selected && 'bg-white/10')}
                  >
                    <IconBox option={o} />
                    <span className="min-w-0 flex-1">
                      <span className="block text-sm font-medium">{o.label}</span>
                      <span className="block truncate text-[11px] text-muted-foreground">{o.hint}</span>
                    </span>
                    {selected && <Check className="h-4 w-4 shrink-0 text-primary" />}
                  </button>
                )
              })}
              {section.key === 'model' && (
                <>
                  <button
                    type="button"
                    onClick={() => setShowMore((v) => !v)}
                    className="mt-1 flex w-full items-center justify-between rounded-lg px-2 py-2 text-left text-xs text-muted-foreground transition-colors hover:bg-white/5"
                  >
                    {showMore ? 'Sembunyikan model lainnya' : 'Model lainnya'}
                    <ChevronUp className={cn('h-3.5 w-3.5 transition-transform', !showMore && 'rotate-180')} />
                  </button>
                  <div className="mt-1 space-y-1.5 rounded-lg border border-border/60 px-2 py-2">
                    <p className="text-[10px] text-muted-foreground">
                      Custom / API (bukan hanya Max). Contoh: claude-sonnet-4-20250514
                    </p>
                    <div className="flex gap-1.5">
                      <input
                        type="text"
                        spellCheck={false}
                        autoComplete="off"
                        placeholder="claude-…"
                        value={customModel}
                        onChange={(e) => setCustomModel(e.target.value.trim())}
                        className="min-w-0 flex-1 rounded-md border border-border bg-background/60 px-2 py-1.5 font-mono text-[11px] outline-none focus:border-primary"
                      />
                      <button
                        type="button"
                        disabled={!customOk}
                        onClick={() => {
                          const v = customModel.trim()
                          if (!isValidModelId(v)) return
                          onChange({ ...prefs, model: v })
                          setView(null)
                        }}
                        className="shrink-0 rounded-md bg-primary/20 px-2.5 py-1.5 text-[11px] font-medium text-primary disabled:opacity-40"
                      >
                        Pakai
                      </button>
                    </div>
                  </div>
                </>
              )}
            </>
          )}
        </div>
      )}
    </div>
  )
}
