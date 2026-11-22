import { useCallback, useEffect, useState } from 'react'

export type View = 'chat' | 'profile' | 'skills'

const PROFILE_HASH = '#/profile'
const SKILLS_HASH = '#/skills'

const read = (): View => {
  const h = window.location.hash
  if (h === PROFILE_HASH) return 'profile'
  if (h === SKILLS_HASH || h.startsWith('#/skills')) return 'skills'
  return 'chat'
}

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
    } else if (v === 'skills') {
      window.location.hash = SKILLS_HASH
    } else if (window.location.hash) {
      history.pushState(null, '', window.location.pathname + window.location.search)
      setView('chat')
    } else {
      setView('chat')
    }
  }, [])

  return [view, go]
}
