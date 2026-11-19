import { Download } from 'lucide-react'
import type { Attachment } from '@/api/types'
import { uploadUrl } from '@/api/uploads'
import { AttachmentIcon } from '@/components/chat/AttachmentIcon'
import { cn } from '@/lib/utils'

/** Lampiran pada bubble pesan: gambar tampil sebagai thumbnail, lainnya sebagai chip yang bisa diunduh. */
export function MessageAttachments({ attachments, isUser }: { attachments: Attachment[]; isUser: boolean }) {
  if (attachments.length === 0) return null
  return (
    <div className={cn('flex max-w-full flex-wrap gap-2', isUser ? 'justify-end' : 'justify-start')}>
      {attachments.map((a, i) => {
        const href = a.id ? uploadUrl(a.id) : undefined
        if (a.kind === 'image' && href) {
          return (
            <a
              key={`${a.id}-${i}`}
              href={href}
              target="_blank"
              rel="noopener noreferrer"
              title={a.name}
              className="block overflow-hidden rounded-xl border border-border/70 bg-card"
            >
              <img src={href} alt={a.name} loading="lazy" className="max-h-40 max-w-[16rem] object-cover" />
            </a>
          )
        }
        const body = (
          <>
            <AttachmentIcon kind={a.kind} className="h-8 w-8" />
            <span className="min-w-0 flex-1 truncate text-xs font-medium" title={a.name}>
              {a.name}
            </span>
            {href && <Download className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />}
          </>
        )
        const cls = 'flex w-52 max-w-full items-center gap-2 rounded-xl border border-border/70 bg-card/80 p-1.5 pr-2.5'
        return href ? (
          <a key={`${a.id}-${i}`} href={href} download={a.name} className={cn(cls, 'transition-colors hover:bg-card')}>
            {body}
          </a>
        ) : (
          <div key={`${a.name}-${i}`} className={cls}>
            {body}
          </div>
        )
      })}
    </div>
  )
}
