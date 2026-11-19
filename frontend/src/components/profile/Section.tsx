import type { LucideIcon } from 'lucide-react'
import type { ReactNode } from 'react'
import { cn } from '@/lib/utils'

type Props = {
  icon: LucideIcon
  tone: string
  title: string
  description?: string
  children: ReactNode
  /** Elemen di kanan judul (mis. badge status). */
  aside?: ReactNode
}

/** Kartu bagian pada halaman profil. */
export function Section({ icon: Icon, tone, title, description, aside, children }: Props) {
  return (
    <section className="rounded-2xl border border-border/70 bg-card/70 p-4 sm:p-5">
      <header className="flex items-start gap-3">
        <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-background/60">
          <Icon className={cn('h-4.5 w-4.5', tone)} />
        </span>
        <div className="min-w-0 flex-1">
          <h2 className="text-sm font-semibold tracking-tight">{title}</h2>
          {description && <p className="mt-0.5 text-xs text-muted-foreground">{description}</p>}
        </div>
        {aside}
      </header>
      <div className="mt-4">{children}</div>
    </section>
  )
}

/** Baris "label — nilai" untuk informasi hanya-baca. */
export function InfoRow({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex flex-col gap-0.5 py-2 sm:flex-row sm:items-baseline sm:gap-4">
      <dt className="w-40 shrink-0 text-xs text-muted-foreground">{label}</dt>
      <dd className="min-w-0 break-words text-sm">{children}</dd>
    </div>
  )
}
