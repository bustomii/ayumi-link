import { useCallback, useEffect, useRef, useState } from 'react'
import type { Attachment, AttachmentKind } from '@/api/types'
import { deleteUpload, uploadFile, type UploadInfo, type UploadLimits } from '@/api/uploads'
import { kindOfFile, screenFiles } from '@/lib/attachments'
import { describeError } from '@/lib/errors'

export type PendingAttachment = {
  key: string
  name: string
  size: number
  kind: AttachmentKind
  status: 'uploading' | 'ready' | 'error'
  /** 0..1 saat mengunggah */
  progress: number
  info?: UploadInfo
  error?: string
}

let counter = 0

/**
 * Lampiran yang sedang disiapkan di kotak pesan: validasi awal, unggah dengan progres,
 * hapus dari server bila dibatalkan atau tidak jadi dikirim.
 */
export function useAttachments(limits: UploadLimits, onProblems: (problems: string[]) => void) {
  const [items, setItems] = useState<PendingAttachment[]>([])
  const itemsRef = useRef(items)
  useEffect(() => {
    itemsRef.current = items
  })
  const controllers = useRef(new Map<string, AbortController>())

  const patch = useCallback((key: string, p: Partial<PendingAttachment>) => {
    setItems((list) => list.map((i) => (i.key === key ? { ...i, ...p } : i)))
  }, [])

  const add = useCallback(
    (files: File[]) => {
      if (files.length === 0) return
      const { accepted, problems } = screenFiles(files, itemsRef.current.length, limits)
      if (problems.length) onProblems(problems)
      if (accepted.length === 0) return

      const fresh: PendingAttachment[] = accepted.map((f) => ({
        key: `att-${++counter}`,
        name: f.name,
        size: f.size,
        kind: kindOfFile(f.name, limits) ?? 'pdf',
        status: 'uploading',
        progress: 0,
      }))
      setItems((list) => [...list, ...fresh])

      accepted.forEach((file, i) => {
        const key = fresh[i].key
        const ctrl = new AbortController()
        controllers.current.set(key, ctrl)
        uploadFile(file, (p) => patch(key, { progress: p }), ctrl.signal)
          .then((info) => patch(key, { status: 'ready', progress: 1, info, name: info.name, kind: info.kind }))
          .catch((e) => {
            if (e instanceof DOMException && e.name === 'AbortError') return
            patch(key, { status: 'error', error: describeError(e, 'Mengunggah file').message })
          })
          .finally(() => controllers.current.delete(key))
      })
    },
    [limits, onProblems, patch],
  )

  /** Batalkan/buang satu lampiran (dihapus dari server bila sudah terunggah). */
  const remove = useCallback((key: string) => {
    controllers.current.get(key)?.abort()
    const item = itemsRef.current.find((i) => i.key === key)
    if (item?.info) void deleteUpload(item.info.id).catch(() => undefined)
    setItems((list) => list.filter((i) => i.key !== key))
  }, [])

  /** Setelah pesan terkirim: lampiran sudah dipakai, jangan dihapus dari server. */
  const clear = useCallback(() => setItems([]), [])

  /** Buang semua yang belum terkirim (pindah session / komponen ditutup). */
  const discard = useCallback(() => {
    controllers.current.forEach((c) => c.abort())
    controllers.current.clear()
    for (const i of itemsRef.current) if (i.info) void deleteUpload(i.info.id).catch(() => undefined)
    setItems([])
  }, [])

  const ready = items.filter((i) => i.status === 'ready' && i.info)
  const attachments: Attachment[] = ready.map((i) => ({ id: i.info!.id, name: i.info!.name, kind: i.info!.kind }))

  return {
    items,
    add,
    remove,
    clear,
    discard,
    attachments,
    uploading: items.some((i) => i.status === 'uploading'),
    hasError: items.some((i) => i.status === 'error'),
  }
}
