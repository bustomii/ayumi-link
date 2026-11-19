import { Skeleton } from '@/components/ui/skeleton'

/** Kerangka percakapan saat isi session sedang dimuat. */
export function ChatSkeleton() {
  return (
    <div className="flex flex-col gap-5" role="status" aria-label="Memuat percakapan">
      <div className="flex justify-end gap-3">
        <div className="space-y-2">
          <Skeleton className="ml-auto h-3 w-24" />
          <Skeleton className="h-12 w-64 max-w-[70vw] rounded-2xl" />
        </div>
        <Skeleton className="h-9 w-9 shrink-0 rounded-full" />
      </div>
      <div className="flex gap-3">
        <Skeleton className="h-9 w-9 shrink-0 rounded-full" />
        <div className="w-full max-w-[34rem] space-y-2">
          <Skeleton className="h-3 w-28" />
          <Skeleton className="h-4 w-full" />
          <Skeleton className="h-4 w-[92%]" />
          <Skeleton className="h-4 w-[78%]" />
          <Skeleton className="h-4 w-[55%]" />
        </div>
      </div>
      <div className="flex justify-end gap-3">
        <div className="space-y-2">
          <Skeleton className="ml-auto h-3 w-24" />
          <Skeleton className="h-9 w-48 max-w-[60vw] rounded-2xl" />
        </div>
        <Skeleton className="h-9 w-9 shrink-0 rounded-full" />
      </div>
      <div className="flex gap-3">
        <Skeleton className="h-9 w-9 shrink-0 rounded-full" />
        <div className="w-full max-w-[28rem] space-y-2">
          <Skeleton className="h-3 w-28" />
          <Skeleton className="h-4 w-full" />
          <Skeleton className="h-4 w-[70%]" />
        </div>
      </div>
    </div>
  )
}

export function ChatHeaderSkeleton() {
  return (
    <div className="min-w-0 flex-1 space-y-2 py-0.5">
      <Skeleton className="h-4 w-40" />
      <Skeleton className="h-3 w-64 max-w-full" />
    </div>
  )
}
