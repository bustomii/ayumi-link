import { AlertCircle, Loader2, X } from 'lucide-react'
import { uploadUrl } from '@/api/uploads'
import { AttachmentIcon } from '@/components/chat/AttachmentIcon'
import type { PendingAttachment } from '@/hooks/useAttachments'
import { formatSize } from '@/lib/attachments'
import { cn } from '@/lib/utils'

type Props = { items: PendingAttachment[]; onRemove: (key: string) => void }

/** Lampiran yang sedang disiapkan di atas kotak pesan: progres unggah, status, tombol hapus. */
export function AttachmentChips({ items, onRemove }: Props) {
  if (items.length === 0) return null
  return (
    <ul className="mb-2 flex flex-wrap gap-2" aria-label="Lampiran">
      {items.map((i) => (
        <li
          key={i.key}
          className={cn(
            'relative flex w-56 max-w-full items-center gap-2.5 overflow-hidden rounded-xl border bg-card/80 py-1.5 pl-1.5 pr-8',
            i.status === 'error' ? 'border-red-900/60' : 'border-border',
          )}
        >
          {i.status === 'ready' && i.kind === 'image' && i.info ? (
            <img
              src={uploadUrl(i.info.id)}
              alt=""
              className="h-9 w-9 shrink-0 rounded-lg object-cover"
              loading="lazy"
            />
          ) : (
            <AttachmentIcon kind={i.kind} />
          )}
          <div className="min-w-0 flex-1">
            <p className="truncate text-xs font-medium" title={i.name}>
              {i.name}
            </p>
            <p
              className={cn('flex items-center gap-1 truncate text-[11px]', i.status === 'error' ? 'text-red-400' : 'text-muted-foreground')}
              title={i.error}
            >
              {i.status === 'uploading' && (
                <>
                  <Loader2 className="h-3 w-3 shrink-0 animate-spin" />
                  Mengunggah {Math.round(i.progress * 100)}%
                </>
              )}
              {i.status === 'ready' && formatSize(i.size)}
              {i.status === 'error' && (
                <>
                  <AlertCircle className="h-3 w-3 shrink-0" />
                  <span className="truncate">{i.error ?? 'Gagal'}</span>
                </>
              )}
            </p>
          </div>
          {i.status === 'uploading' && (
            <span
              className="absolute inset-x-0 bottom-0 h-0.5 bg-primary transition-[width]"
              style={{ width: `${Math.max(4, Math.round(i.progress * 100))}%` }}
              aria-hidden="true"
            />
          )}
          <button
            type="button"
            onClick={() => onRemove(i.key)}
            aria-label={`Hapus lampiran ${i.name}`}
            title="Hapus lampiran"
            className="absolute right-1.5 top-1.5 flex h-6 w-6 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-white/10 hover:text-foreground"
          >
            <X className="h-3.5 w-3.5" />
          </button>
        </li>
      ))}
    </ul>
  )
}
