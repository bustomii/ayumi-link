import { api, toApiError } from './http'

export type BrowserStatus = { available: boolean; running: boolean; url?: string; title?: string; tabId?: string }

export type BrowserInput =
  | { type: 'click'; x: number; y: number }
  | { type: 'type'; text: string }
  | { type: 'key'; key: string }
  | { type: 'scroll'; x: number; y: number; deltaY: number }
  | { type: 'back' | 'forward' | 'reload' }

export const fetchBrowserStatus = () => api<BrowserStatus>('/api/browser')

export const openBrowser = (
  url: string,
  width: number,
  height: number,
  opts?: { sessionId?: string; tabId?: string },
) =>
  api<BrowserStatus>('/api/browser/open', {
    method: 'POST',
    json: { url, width, height, sessionId: opts?.sessionId, tabId: opts?.tabId },
  })

export const sendBrowserInput = (input: BrowserInput) =>
  api<{ ok: boolean }>('/api/browser/input', { method: 'POST', json: input })

export const closeBrowser = () => api<{ ok: boolean }>('/api/browser/close', { method: 'POST', json: {} })

/** Satu frame JPEG + alamat/judul halaman saat itu. */
export async function fetchBrowserFrame(
  width: number,
  height: number,
  signal?: AbortSignal,
): Promise<{ blob: Blob; url: string; title: string }> {
  const res = await fetch(`/api/browser/frame?w=${width}&h=${height}`, { credentials: 'same-origin', signal })
  if (!res.ok) throw await toApiError(res)
  const dec = (v: string | null) => {
    try {
      return decodeURIComponent(v ?? '')
    } catch {
      return ''
    }
  }
  return {
    blob: await res.blob(),
    url: dec(res.headers.get('X-Browser-Url')),
    title: dec(res.headers.get('X-Browser-Title')),
  }
}
