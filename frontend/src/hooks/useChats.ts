import { useCallback, useRef, useState } from 'react'
import type { Attachment } from '@/api/types'
import { useLatest } from '@/hooks/useLatest'
import { streamChat, type ChatRequest } from '@/api/chat'

export type Chat = {
  user: string
  assistant: string
  tools: string[]
  startedAt: string
  attachments: Attachment[]
}

/**
 * Chat yang sedang berjalan, per kunci (id session, atau kunci draft untuk session baru).
 * Chat tetap berjalan di background walau pengguna pindah ke session lain.
 */
export function useChats(onError?: (e: unknown) => void) {
  const [chats, setChats] = useState<Record<string, Chat>>({})
  const chatsRef = useLatest(chats)
  const onErrorRef = useLatest(onError)
  const aborts = useRef<Record<string, AbortController>>({})

  /** Mengirim pesan; resolve dengan id session (kosong bila gagal sebelum session terbentuk). */
  const send = useCallback(async (key: string, req: ChatRequest, attachments: Attachment[] = []): Promise<string> => {
    if (chatsRef.current[key]) return ''
    const ctrl = new AbortController()
    aborts.current[key] = ctrl
    setChats((c) => ({ ...c, [key]: { user: req.message, assistant: '', tools: [], startedAt: new Date().toISOString(), attachments } }))
    const patch = (fn: (c: Chat) => Chat) =>
      setChats((c) => (c[key] ? { ...c, [key]: fn(c[key]) } : c))

    let sid = req.sessionId ?? ''
    // Layar dikunci / aplikasi ditutup saat balasan mengalir memutus koneksi di sisi browser,
    // sementara server tetap menyelesaikan balasan: itu bukan kegagalan yang perlu dipopup.
    let wasHidden = false
    const onVis = () => {
      if (document.visibilityState === 'hidden') wasHidden = true
    }
    document.addEventListener('visibilitychange', onVis)
    try {
      const res = await streamChat(
        req,
        {
          onSession: (i) => {
            sid = i.id
          },
          onDelta: (t) => patch((c) => ({ ...c, assistant: c.assistant + t })),
          onTool: (n) => patch((c) => (c.tools.includes(n) ? c : { ...c, tools: [...c.tools, n] })),
        },
        ctrl.signal,
      )
      sid = res.id
    } catch (e) {
      const aborted = e instanceof DOMException && e.name === 'AbortError'
      const dropped = e instanceof TypeError || (e instanceof Error && e.message.startsWith('Koneksi terputus'))
      if (!aborted && !(wasHidden && dropped)) onErrorRef.current?.(e)
    } finally {
      document.removeEventListener('visibilitychange', onVis)
    }
    return sid
  }, [chatsRef, onErrorRef])

  /** Menandai chat selesai (menghapus bubble sementara). */
  const finish = useCallback((key: string) => {
    setChats(({ [key]: _drop, ...rest }) => rest)
    delete aborts.current[key]
  }, [])

  const stop = useCallback((key: string) => aborts.current[key]?.abort(), [])

  return { chats, send, finish, stop }
}
