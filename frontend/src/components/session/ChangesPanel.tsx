import { ChevronDown, FileDiff, Loader2, RefreshCw } from 'lucide-react'
import { useEffect, useState } from 'react'
import { fetchSessionChangeDiff, fetchSessionChanges } from '@/api/sessions'
import { GitChanges } from '@/components/session/GitChanges'
import type { ChangeDiff, FileChange } from '@/api/types'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import { cn } from '@/lib/utils'

type Props = { sessionId: string }

function statusTone(status: string) {
  switch (status) {
    case 'added':
      return 'text-emerald-400'
    case 'deleted':
      return 'text-red-400'
    case 'modified':
      return 'text-amber-300'
    default:
      return 'text-muted-foreground'
  }
}

function fileLabel(f: FileChange) {
  return f.path.split('/').pop() || f.path
}

function sortByNewest(changes: FileChange[]) {
  return [...changes].sort((a, b) => {
    const ta = a.updatedAt ? new Date(a.updatedAt).getTime() : 0
    const tb = b.updatedAt ? new Date(b.updatedAt).getTime() : 0
    if (tb !== ta) return tb - ta
    return a.path.localeCompare(b.path)
  })
}

/** Waktu edit terakhir untuk daftar changes. */
function formatEditedAt(iso?: string) {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  const diffMs = Date.now() - d.getTime()
  if (diffMs < 0) {
    return d.toLocaleString('id-ID', { day: '2-digit', month: 'short', hour: '2-digit', minute: '2-digit' })
  }
  const sec = Math.floor(diffMs / 1000)
  if (sec < 60) return 'baru saja'
  const min = Math.floor(sec / 60)
  if (min < 60) return `${min} mnt lalu`
  const hr = Math.floor(min / 60)
  if (hr < 24) return `${hr} jam lalu`
  const day = Math.floor(hr / 24)
  if (day < 7) return `${day} hr lalu`
  return d.toLocaleString('id-ID', {
    day: '2-digit',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function DiffView({ diff, loading }: { diff: ChangeDiff | null; loading: boolean }) {
  if (loading) {
    return (
      <p className="flex items-center gap-2 p-3 text-xs text-muted-foreground">
        <Loader2 className="h-3.5 w-3.5 animate-spin" /> Memuat diff…
      </p>
    )
  }
  if (!diff) return null
  return (
    <pre className="whitespace-pre-wrap break-words p-3 font-mono text-[11px] leading-relaxed sm:text-xs">
      {(diff.diff || '(tidak ada perbedaan dengan backup)').split('\n').map((line, i) => (
        <div
          key={i}
          className={cn(
            line.startsWith('+') && !line.startsWith('+++') && 'bg-emerald-500/10 text-emerald-300',
            line.startsWith('-') && !line.startsWith('---') && 'bg-red-500/10 text-red-300',
            (line.startsWith('+++') || line.startsWith('---')) && 'text-muted-foreground',
          )}
        >
          {line || ' '}
        </div>
      ))}
    </pre>
  )
}

function SessionEdits({ sessionId }: Props) {
  const [list, setList] = useState<FileChange[]>([])
  const [active, setActive] = useState<string>('')
  const [diff, setDiff] = useState<ChangeDiff | null>(null)
  const [loading, setLoading] = useState(true)
  const [diffLoading, setDiffLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const activeFile = list.find((f) => f.path === active)

  const load = async () => {
    setLoading(true)
    setError(null)
    try {
      const changes = sortByNewest(await fetchSessionChanges(sessionId))
      setList(changes)
      if (changes.length) {
        const keep = changes.some((c) => c.path === active)
        setActive(keep ? active : changes[0].path)
      } else {
        setActive('')
        setDiff(null)
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Gagal memuat changes')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    setActive('')
    setDiff(null)
    void load()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sessionId])

  useEffect(() => {
    if (!active) return
    let cancelled = false
    setDiffLoading(true)
    void fetchSessionChangeDiff(sessionId, active)
      .then((d) => {
        if (!cancelled) setDiff(d)
      })
      .catch((e) => {
        if (!cancelled) setError(e instanceof Error ? e.message : 'Gagal memuat diff')
      })
      .finally(() => {
        if (!cancelled) setDiffLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [sessionId, active])

  if (loading && list.length === 0) {
    return (
      <div className="flex min-h-0 flex-1 items-center justify-center gap-2 text-sm text-muted-foreground">
        <Loader2 className="h-4 w-4 animate-spin" /> Memuat changes…
      </div>
    )
  }

  if (!loading && list.length === 0) {
    return (
      <div className="flex min-h-0 flex-1 flex-col items-center justify-center gap-3 p-6 text-center">
        <FileDiff className="h-10 w-10 text-muted-foreground/40" />
        <div>
          <p className="text-sm font-medium text-foreground">Tidak ada perubahan</p>
          <p className="mt-1 text-xs text-muted-foreground">Belum ada file berubah di session ini.</p>
        </div>
        <Button variant="outline" size="sm" className="gap-1.5" onClick={() => void load()}>
          <RefreshCw className="h-3.5 w-3.5" />
          Refresh
        </Button>
        {error && <p className="text-xs text-red-400">{error}</p>}
      </div>
    )
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col overflow-hidden md:flex-row ios-tool-pad">
      {/* Mobile / PWA: dropdown */}
      <div className="flex shrink-0 flex-col gap-2 border-b border-border/70 px-3 py-2 md:hidden">
        <div className="flex items-center gap-2">
          <div className="relative min-w-0 flex-1">
            <select
              value={active}
              onChange={(e) => setActive(e.target.value)}
              className="h-10 w-full appearance-none rounded-lg border border-border bg-card/80 py-2 pl-3 pr-9 text-sm outline-none focus:border-primary"
              aria-label="Pilih file perubahan"
            >
              {list.map((f) => (
                <option key={f.path} value={f.path}>
                  {fileLabel(f)} · {f.status}
                  {f.updatedAt ? ` · ${formatEditedAt(f.updatedAt)}` : ''}
                </option>
              ))}
            </select>
            <ChevronDown className="pointer-events-none absolute top-1/2 right-2.5 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          </div>
          <Button variant="ghost" size="icon" className="h-10 w-10 shrink-0" onClick={() => void load()} title="Refresh">
            <RefreshCw className={cn('h-4 w-4', loading && 'animate-spin')} />
          </Button>
        </div>
        <div className="flex min-w-0 flex-col gap-0.5">
          {diff?.absPath && (
            <p className="truncate text-[11px] text-muted-foreground" title={diff.absPath}>
              {diff.absPath}
            </p>
          )}
          {activeFile?.updatedAt && (
            <p className="text-[11px] text-muted-foreground">
              Terakhir diedit {formatEditedAt(activeFile.updatedAt)}
            </p>
          )}
        </div>
      </div>

      {/* Desktop: daftar samping */}
      <div className="hidden min-h-0 w-64 shrink-0 flex-col border-r border-border/70 md:flex">
        <div className="flex items-center justify-between gap-2 border-b border-border/50 px-3 py-2">
          <p className="text-xs font-medium text-muted-foreground">{list.length} file</p>
          <Button variant="ghost" size="icon" className="h-7 w-7" onClick={() => void load()} title="Refresh">
            <RefreshCw className={cn('h-3.5 w-3.5', loading && 'animate-spin')} />
          </Button>
        </div>
        <ScrollArea className="min-h-0 flex-1 overflow-y-auto">
          <ul className="space-y-0.5 p-1.5">
            {list.map((f) => (
              <li key={f.path}>
                <button
                  type="button"
                  onClick={() => setActive(f.path)}
                  className={cn(
                    'flex w-full flex-col rounded-lg px-2.5 py-2 text-left transition-colors',
                    active === f.path ? 'bg-accent/15 ring-1 ring-accent/30' : 'hover:bg-white/5',
                  )}
                >
                  <span className="truncate text-sm font-medium">{fileLabel(f)}</span>
                  <span className="truncate text-[11px] text-muted-foreground">{f.path}</span>
                  <div className="mt-0.5 flex items-center justify-between gap-2">
                    <span className={cn('text-[10px] font-medium uppercase', statusTone(f.status))}>
                      {f.status}
                    </span>
                    {f.updatedAt && (
                      <span className="truncate text-[10px] text-muted-foreground" title={new Date(f.updatedAt).toLocaleString('id-ID')}>
                        {formatEditedAt(f.updatedAt)}
                      </span>
                    )}
                  </div>
                </button>
              </li>
            ))}
          </ul>
        </ScrollArea>
      </div>

      <div className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
        {error && <p className="px-3 py-2 text-xs text-red-400">{error}</p>}
        <div className="hidden shrink-0 border-b border-border/50 px-3 py-2 md:block">
          <p className="truncate text-sm font-medium">{diff?.path ?? active}</p>
          <p className="truncate text-[11px] text-muted-foreground">{diff?.absPath}</p>
          {(diff?.updatedAt || activeFile?.updatedAt) && (
            <p className="mt-0.5 text-[11px] text-muted-foreground">
              Terakhir diedit {formatEditedAt(diff?.updatedAt || activeFile?.updatedAt)}
            </p>
          )}
        </div>
        <ScrollArea className="min-h-0 flex-1 overflow-auto bg-[#0a0b0e]">
          <DiffView diff={diff} loading={diffLoading} />
        </ScrollArea>
      </div>
    </div>
  )
}

/** Tab Changes: perubahan belum ter-commit (git) dan riwayat edit file oleh session ini. */
export function ChangesPanel({ sessionId }: Props) {
  const [view, setView] = useState<'git' | 'edits'>('git')
  return (
    <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
      <div className="flex shrink-0 gap-1 border-b border-border/60 px-3 py-1.5" role="tablist">
        {(
          [
            ['git', 'Belum di-commit'],
            ['edits', 'Edit session'],
          ] as const
        ).map(([id, label]) => (
          <button
            key={id}
            type="button"
            role="tab"
            aria-selected={view === id}
            onClick={() => setView(id)}
            className={cn(
              'rounded-full px-3 py-1 text-xs font-medium',
              view === id ? 'bg-accent/20 text-foreground' : 'text-muted-foreground hover:bg-white/5',
            )}
          >
            {label}
          </button>
        ))}
      </div>
      {view === 'git' ? (
        <div className="ios-tool-pad flex min-h-0 flex-1 flex-col">
          <GitChanges sessionId={sessionId} />
        </div>
      ) : (
        <SessionEdits sessionId={sessionId} />
      )}
    </div>
  )
}
