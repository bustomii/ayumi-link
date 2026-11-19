/**
 * Memastikan server benar-benar tidak terjangkau sebelum error jaringan ditampilkan.
 * Di iOS/PWA, fetch sering gagal ("Load failed") hanya karena halaman baru dibuka lagi dan jaringan
 * belum siap; beberapa detik kemudian server menjawab normal.
 */

const sleep = (ms: number) => new Promise<void>((r) => setTimeout(r, ms))

/** Menunggu tab terlihat (halaman yang sedang di latar belakang tidak punya jaringan yang andal). */
async function untilVisible(maxMs: number) {
  const start = Date.now()
  while (document.visibilityState === 'hidden' && Date.now() - start < maxMs) await sleep(300)
}

async function ping(timeoutMs: number): Promise<boolean> {
  const ctrl = new AbortController()
  const t = setTimeout(() => ctrl.abort(), timeoutMs)
  try {
    const res = await fetch('/api/health', { cache: 'no-store', credentials: 'same-origin', signal: ctrl.signal })
    return res.ok
  } catch {
    return false
  } finally {
    clearTimeout(t)
  }
}

/** true bila server menjawab dalam beberapa percobaan (jeda bertahap ±6 detik). */
export async function serverReachable(): Promise<boolean> {
  await untilVisible(10_000)
  for (const wait of [0, 800, 1500, 3000]) {
    if (wait) await sleep(wait)
    if (await ping(4000)) return true
  }
  return false
}
