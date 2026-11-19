import { Folder, Paperclip, Send, Square, X } from 'lucide-react'
import { useEffect, useRef, useState, type ClipboardEvent } from 'react'
import { fetchDir } from '@/api/sessions'
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
  /** Folder kerja session: dasar daftar folder saat mengetik "/". */
  cwd?: string
  /** Lampiran yang sedang disiapkan (dikelola ChatPanel agar drag-and-drop bisa menambah). */
  attach: ReturnType<typeof useAttachments>
  /** Atribut `accept` file picker. */
  accept: string
  onSend: (text: string) => void
  onStop: () => void
}

export function ChatComposer({ sending, disabled = false, prefs, onPrefs, placeholder, cwd, attach, accept, onSend, onStop }: Props) {
  const [text, setText] = useState('')
  const ref = useRef<HTMLTextAreaElement>(null)
  const fileRef = useRef<HTMLInputElement>(null)
  const locked = sending || disabled
  const [folders, setFolders] = useState<{ name: string; path: string }[]>([])
  const [dirs, setDirs] = useState<{ name: string; path: string }[]>([])
  const [dirsFor, setDirsFor] = useState('')

  // Token "/…" di akhir teks (diawali spasi / awal baris) memicu daftar folder.
  const m = /(?:^|\s)(\/[^\s]*)$/.exec(text)
  const token = m ? m[1] : null
  const parts = token ? token.slice(1).split('/') : []
  const filter = (parts.pop() ?? '').toLowerCase()
  const baseDir = cwd ? [cwd.replace(/\/$/, ''), ...parts.filter(Boolean)].join('/') : ''
  const picking = token !== null && Boolean(cwd) && !locked

  useEffect(() => {
    if (!picking || !baseDir || baseDir === dirsFor) return
    let alive = true
    fetchDir(baseDir)
      .then((d) => {
        if (!alive) return
        setDirs(d.dirs)
        setDirsFor(baseDir)
      })
      .catch(() => {
        if (!alive) return
        setDirs([])
        setDirsFor(baseDir)
      })
    return () => {
      alive = false
    }
  }, [picking, baseDir, dirsFor])

  const matches = picking && dirsFor === baseDir ? dirs.filter((d) => d.name.toLowerCase().includes(filter)).slice(0, 8) : []

  function pickFolder(d: { name: string; path: string }) {
    setFolders((f) => (f.some((x) => x.path === d.path) ? f : [...f, d]))
    setText((t) => t.replace(/(^|\s)\/[^\s]*$/, '$1'))
    ref.current?.focus()
  }

  useEffect(() => {
    const el = ref.current
    if (!el) return
    el.style.height = 'auto'
    el.style.height = `${Math.min(el.scrollHeight, 200)}px`
  }, [text])

  const hasContent = text.trim() !== '' || folders.length > 0 || attach.attachments.length > 0
  const canSend = hasContent && !attach.uploading && !locked

  function submit() {
    if (!canSend) return
    const prefix = folders.map((f) => `[Folder: ${f.path}]`).join('\n')
    onSend([prefix, text.trim()].filter(Boolean).join('\n'))
    setText('')
    setFolders([])
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
      {matches.length > 0 && (
        <ul
          role="listbox"
          aria-label="Pilih folder"
          className="absolute bottom-full left-0 right-0 z-20 mb-2 max-h-56 overflow-y-auto rounded-xl border border-border bg-card/95 p-1 shadow-lg backdrop-blur"
        >
          {matches.map((d) => (
            <li key={d.path}>
              <button
                type="button"
                onMouseDown={(e) => e.preventDefault()}
                onClick={() => pickFolder(d)}
                className="flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-left text-sm hover:bg-white/5"
              >
                <Folder className="h-4 w-4 shrink-0 text-amber-400" />
                <span className="truncate">{d.name}</span>
              </button>
            </li>
          ))}
        </ul>
      )}
      {folders.length > 0 && (
        <ul className="mb-2 flex flex-wrap gap-1.5" aria-label="Folder tujuan">
          {folders.map((f) => (
            <li
              key={f.path}
              title={f.path}
              className="flex max-w-full items-center gap-1 rounded-full border border-border bg-card/80 py-1 pl-2 pr-1 text-xs"
            >
              <Folder className="h-3.5 w-3.5 shrink-0 text-amber-400" />
              <span className="truncate">{f.name}</span>
              <button
                type="button"
                aria-label={`Hapus folder ${f.name}`}
                onClick={() => setFolders((x) => x.filter((y) => y.path !== f.path))}
                className="rounded-full p-0.5 text-muted-foreground hover:bg-white/10 hover:text-foreground"
              >
                <X className="h-3 w-3" />
              </button>
            </li>
          ))}
        </ul>
      )}
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
