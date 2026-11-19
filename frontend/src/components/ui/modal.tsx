import { useEffect, type ReactNode } from 'react'
import { cn } from '@/lib/utils'

type Props = {
  onClose: () => void
  children: ReactNode
  className?: string
  /** Label aksesibilitas dialog. */
  label: string
}

/** Dialog modal: klik latar atau Escape menutup. */
export function Modal({ onClose, children, className, label }: Props) {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => e.key === 'Escape' && onClose()
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [onClose])

  return (
    <div
      className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 p-4 pt-[max(1rem,var(--safe-t))] pb-[max(1rem,var(--safe-b))] pl-[max(1rem,var(--safe-l))] pr-[max(1rem,var(--safe-r))]"
      onClick={onClose}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-label={label}
        className={cn('w-full max-w-md rounded-2xl border border-border bg-card p-5 shadow-xl', className)}
        onClick={(e) => e.stopPropagation()}
      >
        {children}
      </div>
    </div>
  )
}
