import { ArrowLeft, ChevronDown, ChevronRight, Folder, GitBranch, Loader2, Plus, RefreshCw, Trash2 } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import { addRepo, fetchRepoDiff, fetchRepos, removeRepo, type GitRepo } from '@/api/git'
import { NewSessionDialog } from '@/components/sessions/NewSessionDialog'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

const statusTone: Record<string, string> = {
  added: 'text-emerald-400',
  untracked: 'text-emerald-400',
  deleted: 'text-red-400',
  modified: 'text-amber-300',
  renamed: 'text-sky-300',
}

function DiffBlock({ text }: { text: string }) {
  return (
    <pre className="max-h-[60vh] overflow-auto bg-[#0a0b0e] p-3 font-mono text-[11px] leading-relaxed whitespace-pre-wrap break-words">
      {(text || '(tidak ada perbedaan)').split('\n').map((line, i) => (
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

/** Perubahan belum ter-commit per repo git; file yang diedit chat ini ditandai. */
export function GitChanges({ sessionId }: { sessionId: string }) {
  const [repos, setRepos] = useState<GitRepo[]>([])
  const [selected, setSelected] = useState('')
  const [openFile, setOpenFile] = useState('')
  const [diff, setDiff] = useState<string | null>(null)
  const [onlySession, setOnlySession] = useState(false)
  const [loading, setLoading] = useState(true)
  const [adding, setAdding] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      setRepos(await fetchRepos(sessionId))
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Gagal memuat repo git')
    } finally {
      setLoading(false)
    }
  }, [sessionId])

  useEffect(() => {
    setSelected('')
    setOpenFile('')
    void load()
  }, [load])

  const repo = repos.length === 1 ? repos[0] : repos.find((r) => r.path === selected)
  const files = repo ? repo.files.filter((f) => !onlySession || f.fromSession) : []

  async function toggleFile(path: string) {
    if (!repo) return
    if (openFile === path) {
      setOpenFile('')
      return
    }
    setOpenFile(path)
    setDiff(null)
    try {
      setDiff((await fetchRepoDiff(sessionId, repo.path, path)).diff)
    } catch (e) {
      setDiff(e instanceof Error ? e.message : 'Gagal memuat diff')
    }
  }

  async function change(action: () => Promise<unknown>) {
    setError(null)
    try {
      await action()
      await load()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Gagal mengubah daftar repo')
    }
  }

  function remove(r: GitRepo) {
    if (!window.confirm(`Hapus ${r.name} dari daftar Changes? Folder dan file tidak dihapus.`)) return
    setSelected('')
    void change(() => removeRepo(sessionId, r.path))
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
      <div className="flex shrink-0 items-center gap-2 border-b border-border/60 px-3 py-2">
        {repos.length > 1 && repo && (
          <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => setSelected('')} aria-label="Daftar folder git">
            <ArrowLeft className="h-4 w-4" />
          </Button>
        )}
        <div className="min-w-0 flex-1">
          <p className="truncate text-sm font-medium">{repo ? repo.name : `${repos.length} folder git`}</p>
          {repo && (
            <p className="flex items-center gap-1 truncate text-[11px] text-muted-foreground" title={repo.path}>
              {repo.branch && (
                <>
                  <GitBranch className="h-3 w-3 shrink-0" />
                  {repo.branch} ·{' '}
                </>
              )}
              <span className="truncate">{repo.path}</span>
            </p>
          )}
        </div>
        {repo?.setup && (
          <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => remove(repo)} title="Hapus dari daftar">
            <Trash2 className="h-4 w-4 text-red-400" />
          </Button>
        )}
        <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => setAdding(true)} title="Tambah folder git">
          <Plus className="h-4 w-4" />
        </Button>
        <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => void load()} title="Refresh">
          <RefreshCw className={cn('h-4 w-4', loading && 'animate-spin')} />
        </Button>
      </div>

      {error && <p className="shrink-0 px-3 py-2 text-xs text-red-400">{error}</p>}

      <div className="min-h-0 flex-1 overflow-y-auto overscroll-contain">
        {loading && repos.length === 0 && (
          <p className="flex items-center justify-center gap-2 p-6 text-sm text-muted-foreground">
            <Loader2 className="h-4 w-4 animate-spin" /> Memuat…
          </p>
        )}
        {!loading && repos.length === 0 && (
          <p className="p-6 text-center text-xs text-muted-foreground">
            Belum ada repo git. Tambahkan folder git lewat tombol +.
          </p>
        )}

        {!repo &&
          repos.map((r) => (
            <button
              key={r.path}
              type="button"
              onClick={() => setSelected(r.path)}
              className="flex w-full items-center gap-2.5 border-b border-border/40 px-3 py-3 text-left hover:bg-white/5"
            >
              <Folder className="h-4 w-4 shrink-0 text-amber-400" />
              <span className="min-w-0 flex-1">
                <span className="block truncate text-sm font-medium">{r.name}</span>
                <span className="block truncate text-[11px] text-muted-foreground">{r.path}</span>
                <span className="block text-[11px] text-muted-foreground">
                  {r.error ?? `${r.files.length} belum commit · ${r.files.filter((f) => f.fromSession).length} dari chat ini`}
                </span>
              </span>
              <ChevronRight className="h-4 w-4 shrink-0 text-muted-foreground" />
            </button>
          ))}

        {repo && (
          <>
            <label className="flex items-center gap-2 border-b border-border/40 px-3 py-2 text-xs text-muted-foreground">
              <input type="checkbox" checked={onlySession} onChange={(e) => setOnlySession(e.target.checked)} />
              Hanya yang diedit chat ini
            </label>
            {repo.error && <p className="px-3 py-3 text-xs text-red-400">{repo.error}</p>}
            {!repo.error && files.length === 0 && (
              <p className="p-6 text-center text-xs text-muted-foreground">Tidak ada perubahan yang belum di-commit.</p>
            )}
            {files.map((f) => (
              <div key={f.path} className="border-b border-border/40">
                <button
                  type="button"
                  onClick={() => void toggleFile(f.path)}
                  className="flex w-full items-center gap-2 px-3 py-2.5 text-left hover:bg-white/5"
                >
                  {openFile === f.path ? (
                    <ChevronDown className="h-3.5 w-3.5 shrink-0" />
                  ) : (
                    <ChevronRight className="h-3.5 w-3.5 shrink-0" />
                  )}
                  <span className="min-w-0 flex-1 truncate text-sm" title={f.path}>
                    {f.path}
                  </span>
                  {f.fromSession && (
                    <span className="shrink-0 rounded-full bg-sky-200 px-2 py-0.5 text-[10px] font-medium text-sky-950">
                      chat ini
                    </span>
                  )}
                  <span className={cn('shrink-0 text-[10px] font-medium uppercase', statusTone[f.status])}>{f.status}</span>
                </button>
                {openFile === f.path &&
                  (diff === null ? (
                    <p className="flex items-center gap-2 p-3 text-xs text-muted-foreground">
                      <Loader2 className="h-3.5 w-3.5 animate-spin" /> Memuat diff…
                    </p>
                  ) : (
                    <DiffBlock text={diff} />
                  ))}
              </div>
            ))}
          </>
        )}
      </div>

      {adding && (
        <NewSessionDialog
          title="Tambah folder git"
          description="Pilih folder repo git yang dipantau di tab Changes."
          confirmLabel="Tambahkan folder ini"
          recentCwds={[]}
          onCancel={() => setAdding(false)}
          onCreate={(path) => {
            setAdding(false)
            void change(() => addRepo(sessionId, path))
          }}
        />
      )}
    </div>
  )
}
