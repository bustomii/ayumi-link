import { useCallback, useEffect, useRef, useState } from 'react'
import { fetchSessionDetail } from '@/api/sessions'
import { useLatest } from '@/hooks/useLatest'
import type { ContextUsage, Message, Session, SessionDetail } from '@/api/types'

export const PAGE_SIZE = 20

/** Isi percakapan sebuah session: 20 pesan terakhir, lalu pesan lama saat di-scroll ke atas. */
export function useSessionDetail(activeId: string | null, findSession: (id: string) => Session | null) {
  const [session, setSession] = useState<Session | null>(null)
  const [messages, setMessages] = useState<Message[]>([])
  const [hasMore, setHasMore] = useState(false)
  const [total, setTotal] = useState(0)
  const [context, setContext] = useState<ContextUsage | null>(null)
  const [loading, setLoading] = useState(false)
  const [loadingOlder, setLoadingOlder] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const adoptedRef = useRef<string | null>(null)
  const activeRef = useLatest(activeId)
  const findRef = useLatest(findSession)

  const apply = useCallback((data: SessionDetail) => {
    setSession(data.session)
    setMessages(data.messages ?? [])
    setHasMore(Boolean(data.hasMore))
    setTotal(data.total ?? data.messages?.length ?? 0)
    setContext(data.context ?? null)
    setError(null)
  }, [])

  const clear = useCallback(() => {
    setSession(null)
    setMessages([])
    setHasMore(false)
    setTotal(0)
    setContext(null)
    setError(null)
  }, [])

  /** Pakai data yang sudah dimuat pemanggil (setelah chat) tanpa memuat ulang saat activeId berubah. */
  const adopt = useCallback(
    (id: string, data: SessionDetail) => {
      // Hanya perlu menandai bila activeId akan berubah; bila sama, effect tidak berjalan.
      if (id !== activeRef.current) adoptedRef.current = id
      apply(data)
    },
    [activeRef, apply],
  )

  useEffect(() => {
    if (activeId && adoptedRef.current === activeId) {
      adoptedRef.current = null
      return
    }
    if (!activeId) {
      clear()
      return
    }
    let cancelled = false
    setLoading(true)
    clear()
    fetchSessionDetail(activeId, { limit: PAGE_SIZE })
      .then((data) => !cancelled && apply(data))
      .catch((e) => {
        if (cancelled) return
        setError(e instanceof Error ? e.message : 'Gagal memuat session')
        setSession(findRef.current(activeId))
      })
      .finally(() => !cancelled && setLoading(false))
    return () => {
      cancelled = true
    }
  }, [activeId, apply, clear, findRef])

  const loadOlder = useCallback(async () => {
    const before = messages[0]?.id
    if (!activeId || !before || loadingOlder || loading || !hasMore) return
    setLoadingOlder(true)
    try {
      const data = await fetchSessionDetail(activeId, { limit: PAGE_SIZE, before })
      const older = data.messages ?? []
      if (older.length === 0) {
        setHasMore(false)
        return
      }
      setMessages((prev) => {
        const seen = new Set(prev.map((m) => m.id))
        return [...older.filter((m) => !seen.has(m.id)), ...prev]
      })
      setHasMore(Boolean(data.hasMore))
      setTotal((t) => data.total ?? t)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Gagal memuat pesan lama')
    } finally {
      setLoadingOlder(false)
    }
  }, [activeId, hasMore, loading, loadingOlder, messages])

  return { session, messages, hasMore, total, context, loading, loadingOlder, error, loadOlder, adopt, clear }
}
