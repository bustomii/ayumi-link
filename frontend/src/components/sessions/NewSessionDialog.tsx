import { AlertTriangle, ChevronLeft, Folder, FolderOpen } from 'lucide-react'
import { useEffect, useState } from 'react'
import { fetchDir, type DirListing } from '@/api/sessions'
import { Button } from '@/components/ui/button'
import { Modal } from '@/components/ui/modal'
import { Skeleton } from '@/components/ui/skeleton'
import { describeError, type AppError } from '@/lib/errors'

type Props = {
  recentCwds: string[]
  onCancel: () => void
  onCreate: (cwd: string) => void
}

export function NewSessionDialog({ recentCwds, onCancel, onCreate }: Props) {
  const [listing, setListing] = useState<DirListing | null>(null)
  const [error, setError] = useState<AppError | null>(null)
  const [loading, setLoading] = useState(true)
  const [root, setRoot] = useState('')

  function open(path?: string) {
    setLoading(true)
    setError(null)
    fetchDir(path)
      .then((d) => {
        setRoot((r) => r || d.path)
        setListing(d)
      })
      .catch((e) => setError(describeError(e, 'Membuka folder')))
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    open()
  }, [])

  const recents = root ? recentCwds.filter((c) => c === root || c.startsWith(root + '/')).slice(0, 5) : []
  const shown = listing ? (root && listing.path.startsWith(root) ? '~' + listing.path.slice(root.length) : listing.path) : ''

  return (
    <Modal label="Session baru" onClose={onCancel} className="flex max-h-[85vh] flex-col">
      <>
        <p className="text-base font-semibold">Session baru</p>
        <p className="mt-1 text-xs text-muted-foreground">
          Pilih folder kerja tempat Claude dijalankan di Mac.
        </p>

        {recents.length > 0 && (
          <div className="mt-3">
            <p className="mb-1 text-[11px] font-medium text-muted-foreground">Terakhir dipakai</p>
            <div className="flex flex-wrap gap-1.5">
              {recents.map((c) => (
                <button
                  key={c}
                  type="button"
                  title={c}
                  onClick={() => onCreate(c)}
                  className="max-w-full truncate rounded-full border border-border px-2.5 py-1 text-xs hover:bg-white/5"
                >
                  {c.split('/').filter(Boolean).pop()}
                </button>
              ))}
            </div>
          </div>
        )}

        <div className="mt-3 flex items-center gap-1.5">
          <Button
            type="button"
            variant="ghost"
            size="icon"
            disabled={!listing?.parent || loading}
            onClick={() => listing?.parent && open(listing.parent)}
            title="Naik satu folder"
            className="h-8 w-8 shrink-0 rounded-full"
          >
            <ChevronLeft className="h-4 w-4" />
          </Button>
          <p className="min-w-0 flex-1 truncate rounded-lg bg-background/60 px-2.5 py-1.5 text-xs" title={listing?.path}>
            {shown || '…'}
          </p>
        </div>

        <div className="mt-2 min-h-32 flex-1 space-y-0.5 overflow-y-auto">
          {loading && (
            <div className="space-y-1.5 px-2 py-1" role="status" aria-label="Memuat folder">
              {[70, 55, 80, 45, 62].map((w, i) => (
                <div key={i} className="flex items-center gap-2 py-1">
                  <Skeleton className="h-4 w-4 shrink-0" />
                  <Skeleton className="h-3.5" style={{ width: `${w}%` }} />
                </div>
              ))}
            </div>
          )}
          {error && (
            <div role="alert" className="flex gap-2 rounded-lg bg-red-950/30 px-3 py-2.5 text-xs">
              <AlertTriangle className="mt-px h-4 w-4 shrink-0 text-red-400" />
              <div className="space-y-0.5">
                <p className="font-medium text-red-300">{error.title}</p>
                <p className="text-muted-foreground">{error.hint ?? error.message}</p>
              </div>
            </div>
          )}
          {!loading && !error && listing && listing.dirs.length === 0 && (
            <p className="px-2 py-3 text-xs text-muted-foreground">Tidak ada subfolder.</p>
          )}
          {!loading &&
            listing?.dirs.map((d) => (
              <button
                key={d.path}
                type="button"
                onClick={() => open(d.path)}
                className="flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-left text-sm hover:bg-white/5"
              >
                <Folder className="h-4 w-4 shrink-0 text-amber-400" />
                <span className="truncate">{d.name}</span>
              </button>
            ))}
          {listing?.truncated && (
            <p className="px-2 py-2 text-[11px] text-muted-foreground">Daftar dipotong; masuk ke subfolder untuk melihat lainnya.</p>
          )}
        </div>

        <div className="mt-4 flex items-center justify-end gap-2">
          <Button type="button" variant="ghost" onClick={onCancel}>
            Batal
          </Button>
          <Button type="button" disabled={!listing || loading} onClick={() => listing && onCreate(listing.path)}>
            <FolderOpen className="mr-1.5 h-4 w-4" />
            Pilih folder ini
          </Button>
        </div>
      </>
    </Modal>
  )
}
