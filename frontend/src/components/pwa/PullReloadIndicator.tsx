import { Loader2, RefreshCw } from 'lucide-react'
import type { PullReloadState } from '@/hooks/usePwaPullReload'
import { cn } from '@/lib/utils'

const THRESHOLD = 72

/** Indikator tarik-untuk-reload ala mobile (hanya tampil saat gesture). */
export function PullReloadIndicator({ distance, armed, reloading }: PullReloadState) {
  if (distance <= 0 && !reloading) return null

  const progress = Math.min(1, distance / THRESHOLD)
  const y = Math.min(56, 8 + distance * 0.45)

  return (
    <div
      className="pointer-events-none fixed inset-x-0 z-[100] flex justify-center"
      style={{ top: `calc(var(--safe-t, 0px) + ${y}px)` }}
      aria-hidden
    >
      <div
        className={cn(
          'flex h-9 w-9 items-center justify-center rounded-full border border-border/80 bg-card/95 shadow-lg backdrop-blur-sm transition-colors',
          armed || reloading ? 'text-primary' : 'text-muted-foreground',
        )}
        style={{
          opacity: reloading ? 1 : 0.35 + progress * 0.65,
          transform: `scale(${0.75 + progress * 0.25})`,
        }}
      >
        {reloading || armed ? (
          <Loader2 className="h-4 w-4 animate-spin" />
        ) : (
          <RefreshCw className="h-4 w-4" style={{ transform: `rotate(${progress * 180}deg)` }} />
        )}
      </div>
    </div>
  )
}
