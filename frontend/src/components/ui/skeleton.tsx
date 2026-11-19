import type { HTMLAttributes } from 'react'
import { cn } from '@/lib/utils'

/** Placeholder berdenyut untuk konten yang sedang dimuat. */
export function Skeleton({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      aria-hidden="true"
      className={cn('animate-pulse rounded-md bg-white/[0.06]', className)}
      {...props}
    />
  )
}
