import { AuthLayout } from '@/components/auth/AuthLayout'
import { Skeleton } from '@/components/ui/skeleton'

/** Ditampilkan saat status login sedang dicek. */
export function AuthSkeleton() {
  return (
    <AuthLayout title="Memeriksa sesi…">
      <div className="space-y-4" role="status" aria-label="Memeriksa sesi login">
        <Skeleton className="h-3 w-24" />
        <Skeleton className="h-10 w-full rounded-lg" />
        <Skeleton className="h-3 w-20" />
        <Skeleton className="h-10 w-full rounded-lg" />
        <Skeleton className="h-10 w-full rounded-lg" />
      </div>
    </AuthLayout>
  )
}
