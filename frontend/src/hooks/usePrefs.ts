import { useCallback, useEffect, useState } from 'react'
import { fetchPrefs, savePrefs } from '@/api/profile'
import type { Prefs } from '@/lib/chat-options'

const KEY = 'ayumi-link:prefs'
const FALLBACK: Prefs = { permissionMode: 'auto', model: 'claude-sonnet-5-5', effort: 'medium' }

function loadCached(): Prefs {
  try {
    const raw = localStorage.getItem(KEY)
    return raw ? { ...FALLBACK, ...(JSON.parse(raw) as Partial<Prefs>) } : FALLBACK
  } catch {
    return FALLBACK
  }
}

function cache(p: Prefs) {
  try {
    localStorage.setItem(KEY, JSON.stringify(p))
  } catch {
    /* penyimpanan tidak tersedia */
  }
}

/**
 * Preferensi chat default (model, effort, izin tool). Sumber kebenarannya di server supaya sama
 * di semua perangkat; salinan lokal dipakai agar tampil instan sebelum server menjawab.
 */
export function usePrefs(): [Prefs, (p: Prefs) => void] {
  const [prefs, setPrefsState] = useState<Prefs>(loadCached)

  useEffect(() => {
    let cancelled = false
    fetchPrefs()
      .then((p) => {
        if (cancelled) return
        setPrefsState(p)
        cache(p)
      })
      .catch(() => {
        /* tetap pakai salinan lokal */
      })
    return () => {
      cancelled = true
    }
  }, [])

  const setPrefs = useCallback((p: Prefs) => {
    setPrefsState(p)
    cache(p)
    savePrefs(p).catch(() => {
      /* gagal menyimpan ke server: tetap berlaku di perangkat ini */
    })
  }, [])

  return [prefs, setPrefs]
}
