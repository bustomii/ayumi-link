import { Skeleton } from '@/components/ui/skeleton'

/** Kerangka daftar folder + session saat daftar pertama kali dimuat. */
export function SidebarSkeleton() {
  return (
    <div className="space-y-4" role="status" aria-label="Memuat sessions">
      {[3, 2, 2].map((count, g) => (
        <div key={g} className="space-y-2">
          <Skeleton className="h-9 w-full rounded-lg" />
          <div className="space-y-1.5 pl-1">
            {Array.from({ length: count }).map((_, i) => (
              <div key={i} className="flex items-start gap-3 rounded-xl px-3 py-2.5">
                <Skeleton className="mt-0.5 h-8 w-8 shrink-0 rounded-lg" />
                <div className="min-w-0 flex-1 space-y-2 pt-0.5">
                  <Skeleton className="h-3.5 w-[75%]" />
                  <Skeleton className="h-3 w-[45%]" />
                </div>
              </div>
            ))}
          </div>
        </div>
      ))}
    </div>
  )
}
