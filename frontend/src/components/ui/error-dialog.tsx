import {
  AlertTriangle,
  ChevronDown,
  Clock,
  FileQuestion,
  Lightbulb,
  LockKeyhole,
  ServerCrash,
  ShieldAlert,
  Sparkles,
  WifiOff,
  type LucideIcon,
} from 'lucide-react'
import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Modal } from '@/components/ui/modal'
import type { AppError, ErrorKind } from '@/lib/errors'
import { cn } from '@/lib/utils'

const STYLE: Record<ErrorKind, { icon: LucideIcon; tone: string; ring: string }> = {
  network: { icon: WifiOff, tone: 'text-amber-400', ring: 'bg-amber-500/15' },
  auth: { icon: LockKeyhole, tone: 'text-sky-400', ring: 'bg-sky-500/15' },
  forbidden: { icon: ShieldAlert, tone: 'text-orange-400', ring: 'bg-orange-500/15' },
  notfound: { icon: FileQuestion, tone: 'text-violet-400', ring: 'bg-violet-500/15' },
  outdated: { icon: Sparkles, tone: 'text-fuchsia-400', ring: 'bg-fuchsia-500/15' },
  ratelimit: { icon: Clock, tone: 'text-amber-400', ring: 'bg-amber-500/15' },
  server: { icon: ServerCrash, tone: 'text-red-400', ring: 'bg-red-500/15' },
  validation: { icon: AlertTriangle, tone: 'text-amber-400', ring: 'bg-amber-500/15' },
  claude: { icon: AlertTriangle, tone: 'text-red-400', ring: 'bg-red-500/15' },
  unknown: { icon: AlertTriangle, tone: 'text-red-400', ring: 'bg-red-500/15' },
}

type Props = {
  error: AppError
  onClose: () => void
  /** Bila ada, tombol "Coba lagi" ditampilkan. */
  onRetry?: () => void
}

/** Popup error: ikon, penjelasan, saran langkah berikutnya, dan detail teknis yang bisa dibuka. */
export function ErrorDialog({ error, onClose, onRetry }: Props) {
  const [showDetail, setShowDetail] = useState(false)
  const { icon: Icon, tone, ring } = STYLE[error.kind]

  return (
    <Modal label={error.title} onClose={onClose} className="max-w-sm">
      <div role="alert" className="flex flex-col items-center text-center">
        <span className={cn('flex h-12 w-12 items-center justify-center rounded-full', ring)}>
          <Icon className={cn('h-6 w-6', tone)} />
        </span>
        <h2 className="mt-3 text-base font-semibold tracking-tight">{error.title}</h2>
        <p className="mt-1.5 whitespace-pre-line text-sm text-muted-foreground">{error.message}</p>
      </div>

      {error.hint && (
        <div className="mt-4 flex gap-2.5 rounded-lg bg-background/60 p-3 text-left">
          <Lightbulb className="mt-0.5 h-4 w-4 shrink-0 text-amber-400" />
          <p className="text-xs leading-relaxed">{error.hint}</p>
        </div>
      )}

      {error.detail && (
        <div className="mt-3">
          <button
            type="button"
            onClick={() => setShowDetail((v) => !v)}
            aria-expanded={showDetail}
            className="flex w-full items-center justify-center gap-1 text-[11px] text-muted-foreground hover:text-foreground"
          >
            Detail teknis
            <ChevronDown className={cn('h-3.5 w-3.5 transition-transform', showDetail && 'rotate-180')} />
          </button>
          {showDetail && (
            <pre className="mt-2 max-h-32 overflow-auto whitespace-pre-wrap break-words rounded-lg bg-black/30 p-2.5 text-left font-mono text-[11px] text-muted-foreground">
              {error.detail}
            </pre>
          )}
        </div>
      )}

      <div className="mt-5 flex gap-2">
        {onRetry && (
          <Button
            variant="outline"
            className="flex-1"
            onClick={() => {
              onClose()
              onRetry()
            }}
          >
            Coba lagi
          </Button>
        )}
        <Button className="flex-1" onClick={onClose} autoFocus>
          Mengerti
        </Button>
      </div>
    </Modal>
  )
}
