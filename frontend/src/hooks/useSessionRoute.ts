import { useCallback, useEffect, useState } from 'react'

const SESSION_PATH = /^\/s\/([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})\/?$/

function parse(pathname: string): string | null {
  const m = SESSION_PATH.exec(pathname)
  return m ? m[1].toLowerCase() : null
}

/**
 * Session yang terbuka disimpan di URL: `/s/<id>` membuka session itu (tetap terbuka saat
 * reload), `/` berarti tidak ada session terbuka (menu session baru). Tombol back/forward
 * browser ikut bekerja.
 */
export function useSessionRoute() {
  const [id, setId] = useState<string | null>(() => parse(window.location.pathname))

  useEffect(() => {
    // Path yang tidak dikenal dinormalkan ke "/".
    const p = window.location.pathname
    if (p !== '/' && parse(p) === null) window.history.replaceState(null, '', '/')
    const onPop = () => setId(parse(window.location.pathname))
    window.addEventListener('popstate', onPop)
    return () => window.removeEventListener('popstate', onPop)
  }, [])

  const go = useCallback((next: string | null, opts?: { replace?: boolean }) => {
    const url = next ? `/s/${next}` : '/'
    if (window.location.pathname !== url) {
      window.history[opts?.replace ? 'replaceState' : 'pushState'](null, '', url)
    }
    setId(next)
  }, [])

  return [id, go] as const
}
