import { Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Modal } from '@/components/ui/modal'
import { cn } from '@/lib/utils'

type Props = {
  title: string
  description: string
  confirmLabel: string
  /** danger = tombol merah. */
  tone?: 'default' | 'danger'
  busy?: boolean
  onConfirm: () => void
  onCancel: () => void
}

export function ConfirmDialog({ title, description, confirmLabel, tone = 'default', busy, onConfirm, onCancel }: Props) {
  return (
    <Modal label={title} onClose={onCancel} className="max-w-sm">
      <p className="text-base font-semibold">{title}</p>
      <p className="mt-2 text-sm text-muted-foreground">{description}</p>
      <div className="mt-5 flex justify-end gap-2">
        <Button variant="ghost" onClick={onCancel} disabled={busy}>
          Batal
        </Button>
        <Button
          onClick={onConfirm}
          disabled={busy}
          className={cn(tone === 'danger' && 'bg-red-600 text-white hover:bg-red-500')}
        >
          {busy && <Loader2 className="h-4 w-4 animate-spin" />}
          {confirmLabel}
        </Button>
      </div>
    </Modal>
  )
}
