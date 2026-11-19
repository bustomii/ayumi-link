import { useCallback, useEffect, useState } from 'react'
import { fetchSessions } from '@/api/sessions'
import type { Session } from '@/api/types'

/** Daftar session. `onFirstLoad` dipanggil sekali dengan hasil pertama (mis. memilih session awal). */
export function useSessionList(onFirstLoad?: (sessions: Session[]) => void) {
  const [sessions, setSessions] = useState<Session[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<unknown>(null)

  const reload = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const data = await fetchSessions()
      setSessions(data)
      return data
    } catch (e) {
      setError(e)
      return null
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void reload().then((data) => data && onFirstLoad?.(data))
    // hanya sekali saat mount
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  return { sessions, loading, error, reload }
}
