import { Link2 } from 'lucide-react'
import type { ReactNode } from 'react'

type Props = { title: string; description?: string; children: ReactNode }

/** Kerangka layar auth: kartu di tengah dengan logo. */
export function AuthLayout({ title, description, children }: Props) {
  return (
    <main className="app-shell flex items-center justify-center overflow-y-auto overscroll-contain bg-[radial-gradient(ellipse_at_top,_#1a1d27_0%,_#0c0d10_55%)] p-4 pt-[max(1rem,var(--safe-t))] pb-[max(1rem,var(--safe-b))]">
      <div className="w-full max-w-sm">
        <div className="mb-6 flex items-center justify-center gap-2.5">
          <span className="flex h-9 w-9 items-center justify-center rounded-xl bg-primary text-primary-foreground shadow-lg shadow-primary/20">
            <Link2 className="h-5 w-5" />
          </span>
          <span className="text-lg font-semibold tracking-tight">Ayumi Link</span>
        </div>
        <div className="rounded-2xl border border-border/70 bg-card/80 p-6 shadow-xl backdrop-blur">
          <h1 className="text-base font-semibold tracking-tight">{title}</h1>
          {description && <p className="mt-1 text-sm text-muted-foreground">{description}</p>}
          <div className="mt-5">{children}</div>
        </div>
      </div>
    </main>
  )
}
