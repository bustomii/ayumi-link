import { toApiError } from './http'

export type ChatRequest = {
  sessionId?: string
  cwd?: string
  message: string
  permissionMode?: string
  model?: string
  effort?: string
  /** id lampiran yang sudah diunggah */
  attachments?: string[]
}

export type ChatHandlers = {
  onSession?: (info: { id: string; cwd: string }) => void
  onDelta?: (text: string) => void
  onTool?: (name: string) => void
}

/** Kirim pesan ke Claude lokal lewat SSE. Resolve saat selesai, reject bila error. */
export async function streamChat(
  req: ChatRequest,
  handlers: ChatHandlers,
  signal?: AbortSignal,
): Promise<{ id: string }> {
  const res = await fetch('/api/chat', {
    method: 'POST',
    credentials: 'same-origin',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
    signal,
  })
  if (!res.ok || !res.body) throw await toApiError(res, 'POST')

  const reader = res.body.getReader()
  const decoder = new TextDecoder()
  let buf = ''
  let sessionId = req.sessionId ?? ''
  let finished = false

  const dispatch = (event: string, raw: string) => {
    let data: Record<string, string>
    try {
      data = JSON.parse(raw)
    } catch {
      return
    }
    switch (event) {
      case 'session':
        sessionId = data.id
        handlers.onSession?.({ id: data.id, cwd: data.cwd })
        break
      case 'delta':
        handlers.onDelta?.(data.text ?? '')
        break
      case 'tool':
        handlers.onTool?.(data.name ?? '')
        break
      case 'done':
        finished = true
        break
      case 'error':
        throw new Error(data.message || 'Claude error')
    }
  }

  for (;;) {
    const { value, done } = await reader.read()
    if (done) break
    buf += decoder.decode(value, { stream: true })
    let idx: number
    while ((idx = buf.indexOf('\n\n')) >= 0) {
      const block = buf.slice(0, idx)
      buf = buf.slice(idx + 2)
      let event = 'message'
      let data = ''
      for (const line of block.split('\n')) {
        if (line.startsWith('event: ')) event = line.slice(7)
        else if (line.startsWith('data: ')) data += line.slice(6)
      }
      if (data) dispatch(event, data)
    }
  }
  if (!finished) throw new Error('Koneksi terputus sebelum Claude selesai')
  return { id: sessionId }
}
