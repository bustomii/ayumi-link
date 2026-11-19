import { useCallback, useEffect, useState } from 'react'
import { fetchAuthStatus, logout as apiLogout } from '@/api/auth'
import { AUTH_REQUIRED_EVENT } from '@/api/http'

export type AuthState =
  | { status: 'loading' }
  | { status: 'error'; message: string }
  | { status: 'setup_required' }
  | { status: 'logged_out' }
  | { status: 'authenticated'; username: string }

const HINT_KEY = 'ayumi-link:auth-hint'

/** Nama pengguna dari login terakhir (bukan rahasia; sesi sebenarnya ada di cookie HttpOnly). */
function readHint(): string | null {
  try {
    const raw = localStorage.getItem(HINT_KEY)
    const name = raw ? (JSON.parse(raw) as { username?: unknown }).username : null
    return typeof name === 'string' && name ? name : null
  } catch {
    return null
  }
}

function writeHint(username: string | null) {
  try {
    if (username) localStorage.setItem(HINT_KEY, JSON.stringify({ username }))
    else localStorage.removeItem(HINT_KEY)
  } catch {
    /* penyimpanan tidak tersedia */
  }
}

export function useAuth() {
  // Bila sebelumnya sudah login, langsung tampilkan aplikasi (tanpa layar "memeriksa sesi")
  // sementara server memverifikasi di belakang. Bila ternyata sesi habis, pindah ke login.
  const [state, setState] = useState<AuthState>(() => {
    const hint = readHint()
    return hint ? { status: 'authenticated', username: hint } : { status: 'loading' }
  })

  const refresh = useCallback(async () => {
    try {
      const s = await fetchAuthStatus()
      if (s.state === 'authenticated') {
        writeHint(s.username)
        setState((prev) =>
          prev.status === 'authenticated' && prev.username === s.username
            ? prev
            : { status: 'authenticated', username: s.username },
        )
      } else {
        writeHint(null)
        setState({ status: s.state })
      }
    } catch (e) {
      // Gangguan jaringan sesaat tidak boleh mengeluarkan pengguna yang sedang memakai aplikasi.
      setState((prev) =>
        prev.status === 'authenticated'
          ? prev
          : { status: 'error', message: e instanceof Error ? e.message : 'Tidak bisa menghubungi server' },
      )
    }
  }, [])

  useEffect(() => {
    void refresh()
  }, [refresh])

  // Request terproteksi yang ditolak (sesi habis/dicabut) mengembalikan ke layar login.
  useEffect(() => {
    const onRequired = () => void refresh()
    window.addEventListener(AUTH_REQUIRED_EVENT, onRequired)
    return () => window.removeEventListener(AUTH_REQUIRED_EVENT, onRequired)
  }, [refresh])

  const logout = useCallback(async () => {
    writeHint(null)
    try {
      await apiLogout()
    } finally {
      await refresh()
    }
  }, [refresh])

  const retry = useCallback(() => {
    setState({ status: 'loading' })
    void refresh()
  }, [refresh])

  return { state, refresh, logout, retry }
}
