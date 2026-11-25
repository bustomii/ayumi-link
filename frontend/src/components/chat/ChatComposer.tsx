import { Bot, Folder, Paperclip, Send, Square, Wrench } from 'lucide-react'
import { useCallback, useEffect, useRef, useState, type ClipboardEvent, type KeyboardEvent, type MouseEvent } from 'react'
import { fetchDir } from '@/api/sessions'
import { AttachmentChips } from '@/components/chat/AttachmentChips'
import { tagPrefix, tagStyles, type TagKind } from '@/components/chat/ChatTag'
import { SettingsMenu } from '@/components/chat/SettingsMenu'
import { Button } from '@/components/ui/button'
import type { useAttachments } from '@/hooks/useAttachments'
import type { Prefs } from '@/lib/chat-options'
import { tokenFor } from '@/lib/chat-tags'
import { useMentionIndex } from '@/lib/mention-index'

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

type Trigger = { prefix: '@' | '#' | '/'; raw: string; node: Text; start: number; end: number }
type Dir = { name: string; path: string }

/** Teks yang dikirim: teks biasa + token untuk tiap badge (@agent, #skill, [Folder: path]). */
function serialize(root: HTMLElement): string {
  let out = ''
  let afterChip = false
  const addText = (t: string) => {
    if (!t) return
    if (afterChip && !/^[\s.,;:!?)\]}]/.test(t)) out += ' '
    afterChip = false
    out += t
  }
  const walk = (n: Node) => {
    if (n.nodeType === Node.TEXT_NODE) {
      addText((n.textContent ?? '').replace(/​/g, ''))
      return
    }
    if (!(n instanceof HTMLElement)) return
    const kind = n.dataset.kind as TagKind | undefined
    if (kind) {
      if (out && !/[\s([{]$/.test(out)) out += ' '
      out += tokenFor(kind, n.dataset.value ?? '')
      afterChip = true
      return
    }
    if (n.tagName === 'BR') {
      addText('\n')
      return
    }
    const block = n.tagName === 'DIV' || n.tagName === 'P'
    if (block && out && !out.endsWith('\n')) out += '\n'
    n.childNodes.forEach(walk)
  }
  root.childNodes.forEach(walk)
  return out
}

function buildChip(kind: TagKind, value: string, label: string): HTMLElement {
  const el = document.createElement('span')
  el.contentEditable = 'false'
  el.dataset.kind = kind
  el.dataset.value = value
  el.title = kind === 'folder' ? value : `${tagPrefix[kind]}${label}`
  el.className = `mx-0.5 inline-flex max-w-full items-center rounded-full py-0.5 pr-1 pl-2 align-baseline text-xs font-medium ${tagStyles[kind]}`
  const text = document.createElement('span')
  text.className = 'truncate'
  text.textContent = `${tagPrefix[kind]}${label}`
  const x = document.createElement('span')
  x.dataset.remove = '1'
  x.setAttribute('role', 'button')
  x.setAttribute('aria-label', `Hapus ${tagPrefix[kind]}${label}`)
  x.className = 'ml-1 cursor-pointer rounded-full px-1 leading-none hover:bg-black/15'
  x.textContent = '×'
  el.append(text, x)
  return el
}

export function ChatComposer({
  sending,
  disabled = false,
  prefs,
  onPrefs,
  placeholder,
  cwd,
  attach,
  accept,
  onSend,
  onStop,
}: Props) {
  const editorRef = useRef<HTMLDivElement>(null)
  const fileRef = useRef<HTMLInputElement>(null)
  const locked = sending || disabled
  const { agents, skills } = useMentionIndex()
  const [plain, setPlain] = useState('')
  const [trigger, setTrigger] = useState<Trigger | null>(null)
  const [dirs, setDirs] = useState<Dir[]>([])
  const [dirsFor, setDirsFor] = useState('')

  const sync = useCallback(() => {
    const ed = editorRef.current
    if (!ed) return
    const value = serialize(ed)
    // Browser meninggalkan <br> sisa saat kosong; bersihkan agar placeholder muncul lagi.
    if (value.trim() === '' && !ed.querySelector('[data-kind]') && ed.innerHTML !== '') ed.innerHTML = ''
    setPlain(value)
  }, [])

  const detectTrigger = useCallback(() => {
    const ed = editorRef.current
    const sel = window.getSelection()
    if (!ed || !sel || !sel.isCollapsed || !sel.anchorNode || !ed.contains(sel.anchorNode)) {
      setTrigger(null)
      return
    }
    const node = sel.anchorNode
    if (node.nodeType !== Node.TEXT_NODE) {
      setTrigger(null)
      return
    }
    const before = (node.textContent ?? '').slice(0, sel.anchorOffset)
    const m = /(?:^|[\s([{])([@#/])([^\s]*)$/.exec(before)
    if (!m || (m[1] !== '/' && !/^[\w.-]*$/.test(m[2]))) {
      setTrigger(null)
      return
    }
    setTrigger({
      prefix: m[1] as Trigger['prefix'],
      raw: m[2],
      node: node as Text,
      start: before.length - 1 - m[2].length,
      end: before.length,
    })
  }, [])

  // Daftar folder untuk token "/…".
  const parts = trigger?.prefix === '/' ? trigger.raw.split('/') : []
  const folderFilter = (parts.pop() ?? '').toLowerCase()
  const baseDir = cwd ? [cwd.replace(/\/$/, ''), ...parts.filter(Boolean)].join('/') : ''
  const pickingFolder = trigger?.prefix === '/' && Boolean(cwd) && !locked

  useEffect(() => {
    if (!pickingFolder || !baseDir || baseDir === dirsFor) return
    let alive = true
    fetchDir(baseDir)
      .then((d) => {
        if (!alive) return
        setDirs(Array.isArray(d.dirs) ? d.dirs : [])
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
  }, [pickingFolder, baseDir, dirsFor])

  const mentionFilter = trigger && trigger.prefix !== '/' ? trigger.raw.toLowerCase() : ''
  const matchText = (slug: string, name: string, label?: string) =>
    slug.toLowerCase().includes(mentionFilter) ||
    name.toLowerCase().includes(mentionFilter) ||
    (label ?? '').toLowerCase().includes(mentionFilter)
  const agentHits =
    !locked && trigger?.prefix === '@' ? agents.filter((a) => matchText(a.slug, a.name, a.label)).slice(0, 8) : []
  const skillHits =
    !locked && trigger?.prefix === '#' ? skills.filter((s) => matchText(s.slug, s.name, s.label)).slice(0, 8) : []
  const folderHits =
    pickingFolder && dirsFor === baseDir ? dirs.filter((d) => d.name.toLowerCase().includes(folderFilter)).slice(0, 8) : []

  function insertChip(kind: TagKind, value: string, label: string) {
    const t = trigger
    const ed = editorRef.current
    if (!t || !ed) return
    const range = document.createRange()
    range.setStart(t.node, t.start)
    range.setEnd(t.node, t.end)
    range.deleteContents()
    const chip = buildChip(kind, value, label)
    range.insertNode(chip)
    const space = document.createTextNode(' ')
    chip.after(space)
    const caret = document.createRange()
    caret.setStart(space, 1)
    caret.collapse(true)
    const sel = window.getSelection()
    sel?.removeAllRanges()
    sel?.addRange(caret)
    setTrigger(null)
    sync()
    ed.focus()
  }

  const hasContent = plain.trim() !== '' || attach.attachments.length > 0
  const canSend = hasContent && !attach.uploading && !locked

  function submit() {
    const ed = editorRef.current
    if (!canSend || !ed) return
    onSend(serialize(ed).trim())
    ed.innerHTML = ''
    setPlain('')
    setTrigger(null)
  }

  function onPaste(e: ClipboardEvent<HTMLDivElement>) {
    const files = Array.from(e.clipboardData.files)
    if (files.length > 0) {
      e.preventDefault()
      attach.add(files)
      return
    }
    // Tempel sebagai teks biasa (tanpa format dari sumber lain).
    e.preventDefault()
    document.execCommand('insertText', false, e.clipboardData.getData('text/plain'))
  }

  function onKeyDown(e: KeyboardEvent<HTMLDivElement>) {
    if (e.key === 'Enter' && !e.nativeEvent.isComposing) {
      e.preventDefault()
      if (e.shiftKey) document.execCommand('insertText', false, '\n')
      else submit()
    }
  }

  function onEditorClick(e: MouseEvent<HTMLDivElement>) {
    const x = (e.target as HTMLElement).closest('[data-remove]')
    if (x?.parentElement) {
      x.parentElement.remove()
      sync()
      setTrigger(null)
    }
  }

  // Tinggi editor mengikuti isi; posisi caret menentukan daftar autocomplete.
  useEffect(() => {
    const onSel = () => detectTrigger()
    document.addEventListener('selectionchange', onSel)
    return () => document.removeEventListener('selectionchange', onSel)
  }, [detectTrigger])

  const listClass =
    'absolute right-0 bottom-full left-0 z-20 mb-2 max-h-56 overflow-y-auto rounded-xl border border-border bg-card/95 p-1 shadow-lg backdrop-blur'
  const itemClass = 'flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-left text-sm hover:bg-white/5'

  return (
    <div className="relative mx-auto max-w-3xl">
      {folderHits.length > 0 && (
        <ul role="listbox" aria-label="Pilih folder" className={listClass}>
          {folderHits.map((d) => (
            <li key={d.path}>
              <button
                type="button"
                onMouseDown={(e) => e.preventDefault()}
                onClick={() => insertChip('folder', d.path, d.name)}
                className={itemClass}
              >
                <Folder className="h-4 w-4 shrink-0 text-emerald-400" />
                <span className="truncate">{d.name}</span>
              </button>
            </li>
          ))}
        </ul>
      )}
      {agentHits.length > 0 && (
        <ul role="listbox" aria-label="Pilih agent" className={listClass}>
          {agentHits.map((a) => (
            <li key={a.id}>
              <button
                type="button"
                onMouseDown={(e) => e.preventDefault()}
                onClick={() => insertChip('agent', a.slug, a.slug)}
                className={itemClass}
              >
                <Bot className="h-4 w-4 shrink-0 text-sky-400" />
                <span className="min-w-0 flex-1 truncate">{a.name}</span>
                <span className="text-[11px] text-muted-foreground">@{a.slug}</span>
              </button>
            </li>
          ))}
        </ul>
      )}
      {skillHits.length > 0 && (
        <ul role="listbox" aria-label="Pilih skill" className={listClass}>
          {skillHits.map((s) => (
            <li key={s.id}>
              <button
                type="button"
                onMouseDown={(e) => e.preventDefault()}
                onClick={() => insertChip('skill', s.slug, s.slug)}
                className={itemClass}
              >
                <Wrench className="h-4 w-4 shrink-0 text-violet-400" />
                <span className="min-w-0 flex-1 truncate">{s.name}</span>
                <span className="text-[11px] text-muted-foreground">#{s.slug}</span>
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
            e.target.value = ''
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
        <div
          ref={editorRef}
          role="textbox"
          aria-multiline="true"
          aria-label="Tulis pesan"
          contentEditable={!locked}
          suppressContentEditableWarning
          spellCheck
          data-placeholder={placeholder ?? 'Tulis pesan… @agent #skill /folder'}
          onInput={() => {
            sync()
            detectTrigger()
          }}
          onKeyDown={onKeyDown}
          onKeyUp={detectTrigger}
          onClick={onEditorClick}
          onPaste={onPaste}
          className="chat-editor max-h-[200px] min-h-10 min-w-0 flex-1 overflow-y-auto rounded-xl border border-border bg-card/70 px-3 py-2 text-base leading-6 break-words whitespace-pre-wrap outline-none focus:border-primary md:text-sm aria-disabled:opacity-60"
          aria-disabled={locked}
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
