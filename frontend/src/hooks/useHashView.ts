import { useCallback, useEffect, useState } from 'react'

export type View = 'chat' | 'profile'

const PROFILE_HASH = '#/profile'

const read = (): View => (window.location.hash === PROFILE_HASH ? 'profile' : 'chat')

/** Tampilan aktif disimpan di hash URL, jadi tombol back browser dan tautan langsung berfungsi. */
export function useHashView(): [View, (v: View) => void] {
  const [view, setView] = useState<View>(read)

  useEffect(() => {
    const onChange = () => setView(read())
    window.addEventListener('hashchange', onChange)
    return () => window.removeEventListener('hashchange', onChange)
  }, [])

  const go = useCallback((v: View) => {
    if (v === 'profile') {
      window.location.hash = PROFILE_HASH
    } else if (window.location.hash) {
      // bersihkan hash tanpa menambah riwayat dan tanpa loncat scroll
      history.pushState(null, '', window.location.pathname + window.location.search)
      setView('chat')
    }
  }, [])

  return [view, go]
}
