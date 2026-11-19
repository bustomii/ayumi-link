import { Paperclip, Send, Square } from 'lucide-react'
import { useEffect, useRef, useState, type ClipboardEvent } from 'react'
import { AttachmentChips } from '@/components/chat/AttachmentChips'
import { SettingsMenu } from '@/components/chat/SettingsMenu'
import { Button } from '@/components/ui/button'
import type { useAttachments } from '@/hooks/useAttachments'
import type { Prefs } from '@/lib/chat-options'

type Props = {
  sending: boolean
  disabled?: boolean
  prefs: Prefs
  onPrefs: (p: Prefs) => void
  placeholder?: string
  /** Lampiran yang sedang disiapkan (dikelola ChatPanel agar drag-and-drop bisa menambah). */
  attach: ReturnType<typeof useAttachments>
  /** Atribut `accept` file picker. */
  accept: string
  onSend: (text: string) => void
  onStop: () => void
}

export function ChatComposer({ sending, disabled = false, prefs, onPrefs, placeholder, attach, accept, onSend, onStop }: Props) {
  const [text, setText] = useState('')
  const ref = useRef<HTMLTextAreaElement>(null)
  const fileRef = useRef<HTMLInputElement>(null)
  const locked = sending || disabled

  useEffect(() => {
    const el = ref.current
    if (!el) return
    el.style.height = 'auto'
    el.style.height = `${Math.min(el.scrollHeight, 200)}px`
  }, [text])

  const hasContent = text.trim() !== '' || attach.attachments.length > 0
  const canSend = hasContent && !attach.uploading && !locked

  function submit() {
    if (!canSend) return
    onSend(text.trim())
    setText('')
  }

  // Gambar yang ditempel (mis. screenshot) dilampirkan; teks biasa diproses seperti biasa.
  function onPaste(e: ClipboardEvent<HTMLTextAreaElement>) {
    const files = Array.from(e.clipboardData.files)
    if (files.length > 0) {
      e.preventDefault()
      attach.add(files)
    }
  }

  return (
    <div className="relative mx-auto max-w-3xl">
      <AttachmentChips items={attach.items} onRemove={attach.remove} />
      <div className="flex items-end gap-2">
        <input
          ref={fileRef}
          type="file"
          multiple
          hidden
          accept={accept}
          onChange={(e) => {
            attach.add(Array.from(e.target.files ?? []))
            e.target.value = '' // izinkan memilih file yang sama lagi
          }}
        />
        <Button
          type="button"
          variant="outline"
          size="icon"
          className="h-10 w-10 shrink-0 rounded-full"
          title="Lampirkan file (gambar, PDF, Excel, Word)"
          aria-label="Lampirkan file"
          disabled={locked}
          onClick={() => fileRef.current?.click()}
        >
          <Paperclip className="h-4 w-4" />
        </Button>
        <textarea
          ref={ref}
          value={text}
          rows={1}
          disabled={locked}
          placeholder={placeholder ?? 'Tulis pesan ke Claude…'}
          onChange={(e) => setText(e.target.value)}
          onPaste={onPaste}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && !e.shiftKey && !e.nativeEvent.isComposing) {
              e.preventDefault()
              submit()
            }
          }}
          className="min-h-10 flex-1 resize-none rounded-xl border border-border bg-card/70 px-3 py-2 text-sm outline-none placeholder:text-muted-foreground focus:border-primary disabled:opacity-60"
        />
        <SettingsMenu prefs={prefs} disabled={locked} onChange={onPrefs} />
        {sending ? (
          <Button size="icon" variant="outline" onClick={onStop} title="Hentikan" className="rounded-full">
            <Square className="h-4 w-4" />
          </Button>
        ) : (
          <Button
            size="icon"
            onClick={submit}
            disabled={!canSend}
            title={attach.uploading ? 'Menunggu unggahan selesai' : 'Kirim'}
            className="rounded-full"
          >
            <Send className="h-4 w-4" />
          </Button>
        )}
      </div>
    </div>
  )
}
