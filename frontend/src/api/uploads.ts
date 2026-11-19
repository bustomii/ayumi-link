import { ApiError, AUTH_REQUIRED_EVENT, api } from './http'
import type { AttachmentKind } from './types'

export type UploadInfo = {
  id: string
  name: string
  kind: AttachmentKind
  size: number
  mime: string
  createdAt: string
}

export type UploadLimits = {
  maxBytes: number
  maxFiles: number
  types: { kind: AttachmentKind; label: string; exts: string[] }[]
}

export const fetchUploadLimits = () => api<UploadLimits>('/api/uploads/limits')

export const deleteUpload = (id: string) => api<{ ok: true }>(`/api/uploads/${encodeURIComponent(id)}`, { method: 'DELETE' })

/** URL file lampiran (butuh cookie sesi; gambar tampil inline). */
export const uploadUrl = (id: string) => `/api/uploads/${encodeURIComponent(id)}/file`

/**
 * Unggah satu file dengan laporan progres (0..1). Memakai XMLHttpRequest karena fetch
 * belum bisa melaporkan progres unggahan.
 */
export function uploadFile(file: File, onProgress: (p: number) => void, signal?: AbortSignal): Promise<UploadInfo> {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest()
    xhr.open('POST', '/api/uploads')
    xhr.withCredentials = true
    xhr.responseType = 'json'

    xhr.upload.onprogress = (e) => e.lengthComputable && onProgress(e.loaded / e.total)
    xhr.onerror = () => reject(new TypeError('Network error'))
    xhr.onabort = () => reject(new DOMException('Unggahan dibatalkan', 'AbortError'))
    xhr.onload = () => {
      const body = (xhr.response ?? {}) as { error?: string; code?: string }
      if (xhr.status >= 200 && xhr.status < 300) {
        resolve(xhr.response as UploadInfo)
        return
      }
      if (xhr.status === 401 && (body.code === 'unauthenticated' || body.code === 'setup_required')) {
        window.dispatchEvent(new Event(AUTH_REQUIRED_EVENT))
      }
      reject(
        new ApiError(
          body.error ?? (xhr.status >= 500 ? `Permintaan gagal (${xhr.status})` : 'Unggahan ditolak'),
          xhr.status,
          body.code ?? (body.error ? undefined : 'empty_response'),
          undefined,
          'POST /api/uploads',
        ),
      )
    }

    signal?.addEventListener('abort', () => xhr.abort(), { once: true })
    const form = new FormData()
    form.append('file', file)
    xhr.send(form)
  })
}
