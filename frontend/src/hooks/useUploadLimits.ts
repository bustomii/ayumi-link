import { useEffect, useState } from 'react'
import { fetchUploadLimits, type UploadLimits } from '@/api/uploads'
import { DEFAULT_LIMITS } from '@/lib/attachments'

let cached: UploadLimits | null = null

/** Batas dan jenis lampiran dari server (di-cache); memakai nilai bawaan sampai server menjawab. */
export function useUploadLimits(): UploadLimits {
  const [limits, setLimits] = useState<UploadLimits>(cached ?? DEFAULT_LIMITS)
  useEffect(() => {
    if (cached) return
    let cancelled = false
    fetchUploadLimits()
      .then((l) => {
        cached = l
        if (!cancelled) setLimits(l)
      })
      .catch(() => {
        /* tetap pakai nilai bawaan */
      })
    return () => {
      cancelled = true
    }
  }, [])
  return limits
}
