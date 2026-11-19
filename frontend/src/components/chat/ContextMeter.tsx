import type { ContextUsage } from '@/api/types'
import { cn } from '@/lib/utils'

function fmt(n: number) {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1).replace(/\.0$/, '')}M`
  if (n >= 1000) return `${Math.round(n / 1000)}k`
  return String(n)
}

/** Bilah kuota konteks: token terpakai / jendela konteks, berubah warna saat hampir penuh. */
export function ContextMeter({ usage }: { usage: ContextUsage }) {
  const pct = Math.min(100, Math.max(0, (usage.used / usage.window) * 100))
  const tone = pct >= 90 ? 'bg-red-400' : pct >= 75 ? 'bg-amber-400' : 'bg-sky-400'
  return (
    <div
      className="mt-1 flex items-center gap-2"
      title={`Konteks terpakai ${usage.used.toLocaleString('id-ID')} dari ${usage.window.toLocaleString('id-ID')} token`}
    >
      <div
        role="progressbar"
        aria-label="Kuota konteks"
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={Math.round(pct)}
        className="h-1 w-24 overflow-hidden rounded-full bg-white/10 sm:w-32"
      >
        <div className={cn('h-full rounded-full transition-all', tone)} style={{ width: `${pct}%` }} />
      </div>
      <span className="text-[10px] tabular-nums text-muted-foreground">
        {fmt(usage.used)} / {fmt(usage.window)} · {Math.round(pct)}%
      </span>
    </div>
  )
}
