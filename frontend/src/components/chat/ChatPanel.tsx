import { AlertTriangle, ArrowDown, FileDiff, Globe, Menu, FolderInput, MessageSquare, Paperclip, Plus, Terminal } from 'lucide-react'
import { useCallback, useEffect, useLayoutEffect, useRef, useState, type DragEvent } from 'react'
import type { Message, Session } from '@/api/sessions'
import type { Attachment, ContextUsage } from '@/api/types'
import { AccountMenu } from '@/components/AccountMenu'
import { ChatBubble } from '@/components/chat/ChatBubble'
import { ChatComposer } from '@/components/chat/ChatComposer'
import { ChatHeaderSkeleton, ChatSkeleton } from '@/components/chat/ChatSkeleton'
import { ContextMeter } from '@/components/chat/ContextMeter'
import { BrowserPanel } from '@/components/session/BrowserPanel'
import { ChangesPanel } from '@/components/session/ChangesPanel'
import { TerminalPanel } from '@/components/session/TerminalPanel'
import { useAttachments } from '@/hooks/useAttachments'
import { useErrorDialog } from '@/hooks/useErrorDialog'
import { useMentionIndex } from '@/lib/mention-index'
import { useUploadLimits } from '@/hooks/useUploadLimits'
import { acceptAttr, allowedSummary, formatSize } from '@/lib/attachments'
import type { Prefs } from '@/lib/chat-options'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import { cn } from '@/lib/utils'

type SessionTab = 'chat' | 'changes' | 'terminal' | 'browser'

type Props = {
  session: Session | null
  messages: Message[]
  loading: boolean
  loadingOlder?: boolean
  hasMore?: boolean
  /** Pemakaian jendela konteks giliran terakhir (seperti Claude Desktop). */
  context?: ContextUsage | null
  error: string | null
  onOpenSidebar?: () => void
  onNewSession?: () => void
  /** Folder kerja session hilang: minta pengguna memilih folder baru (fork). */
  onForkRequest?: () => void
  onLoadOlder?: () => void
  sending?: boolean
  onSend?: (text: string, attachments: Attachment[]) => void
  onStop?: () => void
  prefs: Prefs
  onPrefs: (p: Prefs) => void
  username: string
  onOpenProfile: () => void
  onOpenSkills: () => void
  onLogout: () => void
}

export function ChatPanel({
  session,
  messages,
  loading,
  loadingOlder = false,
  hasMore = false,
  context = null,
  error,
  onOpenSidebar,
  onNewSession,
  onForkRequest,
  onLoadOlder,
  sending = false,
  onSend,
  onStop,
  prefs,
  onPrefs,
  username,
  onOpenProfile,
  onOpenSkills,
  onLogout,
}: Props) {
  const scrollRef = useRef<HTMLDivElement>(null)
  const mentionIndex = useMentionIndex()
  const bottomRef = useRef<HTMLDivElement>(null)
  const stickBottomRef = useRef(true)
  const prevScrollHeightRef = useRef(0)
  const prevMsgCountRef = useRef(0)
  const loadingOlderRef = useRef(false)
  const [showJump, setShowJump] = useState(false)

  // ---- lampiran ----
  const limits = useUploadLimits()
  const errors = useErrorDialog()
  const showApp = errors.showApp
  const onProblems = useCallback(
    (problems: string[]) =>
      showApp({
        kind: 'validation',
        title: problems.length > 1 ? 'Beberapa file tidak bisa dilampirkan' : 'File tidak bisa dilampirkan',
        message: problems.join('\n'),
        hint: `Yang boleh: ${allowedSummary(limits)}. Maksimal ${formatSize(limits.maxBytes)} per file dan ${limits.maxFiles} file per pesan.`,
      }),
    [limits, showApp],
  )
  const attach = useAttachments(limits, onProblems)
  const discardAttachments = attach.discard
  const [dragging, setDragging] = useState(false)
  const dragDepth = useRef(0)
  const [tab, setTab] = useState<SessionTab>('chat')

  // Lampiran yang belum terkirim dibuang saat pindah session atau panel ditutup.
  const sessionKey = session?.id ?? ''
  useEffect(() => () => discardAttachments(), [sessionKey, discardAttachments])
  useEffect(() => setTab('chat'), [sessionKey])

  const canTools = Boolean(session?.id)

  const hasFiles = (e: DragEvent) => Array.from(e.dataTransfer.types).includes('Files')
  const dropHandlers = {
    onDragEnter: (e: DragEvent) => {
      if (!hasFiles(e)) return
      e.preventDefault()
      dragDepth.current++
      setDragging(true)
    },
    onDragOver: (e: DragEvent) => {
      if (hasFiles(e)) e.preventDefault()
    },
    onDragLeave: (e: DragEvent) => {
      if (!hasFiles(e)) return
      dragDepth.current = Math.max(0, dragDepth.current - 1)
      if (dragDepth.current === 0) setDragging(false)
    },
    onDrop: (e: DragEvent) => {
      if (!hasFiles(e)) return
      e.preventDefault()
      dragDepth.current = 0
      setDragging(false)
      if (session && !sending) attach.add(Array.from(e.dataTransfer.files))
    },
  }

  useEffect(() => {
    loadingOlderRef.current = loadingOlder
  }, [loadingOlder])

  // New session / initial page: jump to bottom instantly (no animated scroll).
  useLayoutEffect(() => {
    stickBottomRef.current = true
    prevMsgCountRef.current = 0
    const el = scrollRef.current
    if (!el) return
    el.scrollTop = el.scrollHeight
    setShowJump(false)
  }, [session?.id])

  useLayoutEffect(() => {
    const el = scrollRef.current
    if (!el) return

    const prevCount = prevMsgCountRef.current
    const grew = messages.length > prevCount
    const prepended = grew && prevCount > 0 && loadingOlderRef.current

    if (prepended) {
      const delta = el.scrollHeight - prevScrollHeightRef.current
      el.scrollTop = el.scrollTop + delta
    } else if (stickBottomRef.current || prevCount === 0) {
      el.scrollTop = el.scrollHeight
    }

    prevMsgCountRef.current = messages.length
    prevScrollHeightRef.current = el.scrollHeight
  }, [messages, loading])

  function jumpToLatest() {
    const el = scrollRef.current
    if (!el) return
    stickBottomRef.current = true
    el.scrollTo({ top: el.scrollHeight, behavior: 'smooth' })
  }

  function onScroll() {
    const el = scrollRef.current
    if (!el) return
    const distanceFromBottom = el.scrollHeight - el.scrollTop - el.clientHeight
    stickBottomRef.current = distanceFromBottom < 80
    setShowJump(distanceFromBottom > 240)

    if (el.scrollTop < 80 && hasMore && !loadingOlder && !loading) {
      prevScrollHeightRef.current = el.scrollHeight
      onLoadOlder?.()
    }
  }

  if (!session && !loading) {
    return (
      <div className="flex h-full min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
        <header className="flex shrink-0 items-center gap-2 border-b border-border/70 px-3 py-3">
          <Button variant="ghost" size="icon" onClick={onOpenSidebar} aria-label="Buka sessions" title="Sessions">
            <Menu className="h-5 w-5" />
          </Button>
          <span className="min-w-0 flex-1 text-sm font-medium">Ayumi Link</span>
          <AccountMenu
            username={username}
            onOpenProfile={onOpenProfile}
            onOpenSkills={onOpenSkills}
            onLogout={onLogout}
          />
        </header>
        <div className="flex flex-1 items-center justify-center bg-[radial-gradient(ellipse_at_top,_#1a1d27_0%,_#0c0d10_55%)] p-4">
          <div className="max-w-sm rounded-2xl border border-border/60 bg-card/70 px-6 py-8 text-center backdrop-blur">
            <p className="text-lg font-semibold tracking-tight">
              {error ? 'Session tidak bisa dibuka' : 'Mulai percakapan baru'}
            </p>
            <p className="mt-2 text-sm text-muted-foreground first-letter:uppercase">
              {error ?? 'Buat session baru, atau pilih percakapan yang ada dari daftar sessions.'}
            </p>
            <div className="mt-5 flex flex-col gap-2">
              <Button onClick={onNewSession}>
                <Plus className="h-4 w-4" />
                Buat session baru
              </Button>
              <Button variant="outline" onClick={onOpenSidebar}>
                Lihat sessions
              </Button>
            </div>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div
      className="relative flex h-full min-h-0 min-w-0 flex-1 flex-col overflow-hidden bg-[radial-gradient(ellipse_at_top,_#151821_0%,_#0c0d10_50%)]"
      {...dropHandlers}
    >
      {dragging && (
        <div className="pointer-events-none absolute inset-0 z-40 flex items-center justify-center bg-background/80 p-4 backdrop-blur-sm">
          <div className="flex max-w-sm flex-col items-center rounded-2xl border-2 border-dashed border-primary/70 px-8 py-10 text-center">
            <Paperclip className="h-8 w-8 text-primary" />
            <p className="mt-3 text-base font-semibold">Lepaskan untuk melampirkan</p>
            <p className="mt-1 text-xs text-muted-foreground">Gambar, PDF, Excel (.xlsx, .csv), atau Word (.docx)</p>
          </div>
        </div>
      )}
      <header className="flex shrink-0 flex-col border-b border-border/70 bg-background/70 backdrop-blur">
        <div className="flex items-start gap-2 px-3 py-2.5 sm:px-5 sm:py-3">
          <Button
            variant="ghost"
            size="icon"
            className="mt-0.5 shrink-0"
            onClick={onOpenSidebar}
            aria-label="Buka sessions"
            title="Sessions"
          >
            <Menu className="h-5 w-5" />
          </Button>
          {session ? (
            <div className="min-w-0 flex-1">
              <div className="flex min-w-0 items-center gap-2">
                <h1 className="truncate text-sm font-semibold tracking-tight sm:text-base">
                  {session.title}
                </h1>
                {session.live && (
                  <span className="inline-flex shrink-0 items-center gap-1 rounded-full bg-emerald-500/15 px-2 py-0.5 text-[10px] font-medium text-emerald-400">
                    <span className="h-1.5 w-1.5 rounded-full bg-emerald-400" />
                    live
                  </span>
                )}
              </div>
              <p className="truncate text-[11px] text-muted-foreground sm:text-xs">{session.cwd}</p>
              {context && <ContextMeter usage={context} />}
            </div>
          ) : (
            <div className="min-w-0 flex-1">
              <ChatHeaderSkeleton />
            </div>
          )}
          <div className="mt-0.5 shrink-0">
            <AccountMenu
              username={username}
              onOpenProfile={onOpenProfile}
              onOpenSkills={onOpenSkills}
              onLogout={onLogout}
            />
          </div>
        </div>
        {canTools && (
          <nav className="w-full border-t border-border/40" aria-label="Panel session">
            <div className="mx-auto grid w-full grid-cols-4 md:max-w-xl">
              {(
                [
                  { id: 'chat', label: 'Chat', Icon: MessageSquare },
                  { id: 'changes', label: 'Changes', Icon: FileDiff },
                  { id: 'terminal', label: 'Terminal', Icon: Terminal },
                  { id: 'browser', label: 'Browser', Icon: Globe },
                ] as const
              ).map(({ id, label, Icon }) => (
                <button
                  key={id}
                  type="button"
                  onClick={() => setTab(id)}
                  className={cn(
                    'flex w-full flex-col items-center justify-center gap-1 px-2 py-2.5 text-center text-xs font-medium transition-colors sm:flex-row sm:gap-1.5',
                    tab === id
                      ? 'bg-accent/20 text-foreground shadow-[inset_0_-2px_0_0_#3d8bfd]'
                      : 'text-muted-foreground hover:bg-white/5 hover:text-foreground',
                  )}
                >
                  <Icon className="h-4 w-4 shrink-0" />
                  {label}
                </button>
              ))}
            </div>
          </nav>
        )}
      </header>

      {session?.cwdMissing && canTools && (
        <div role="alert" className="flex shrink-0 items-start gap-2.5 border-b border-amber-500/30 bg-amber-500/10 px-3 py-2.5 text-xs">
          <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-amber-400" />
          <div className="min-w-0 flex-1">
            <p className="font-medium text-amber-300">Folder session ini tidak tersedia</p>
            <p className="mt-0.5 break-all text-muted-foreground">{session.cwd}</p>
            <p className="mt-0.5 text-muted-foreground">Folder dihapus, dipindah, atau di luar folder yang diizinkan. Pilih folder lain: session akan di-fork ke sana.</p>
          </div>
          {onForkRequest && (
            <Button type="button" size="sm" onClick={onForkRequest} className="shrink-0 gap-1">
              <FolderInput className="h-3.5 w-3.5" />
              Pilih folder
            </Button>
          )}
        </div>
      )}

      {tab === 'changes' && canTools ? (
        <ChangesPanel sessionId={session!.id} />
      ) : tab === 'browser' && canTools ? (
        <BrowserPanel sessionId={session!.id} />
      ) : tab === 'terminal' && canTools ? (
        <TerminalPanel sessionId={session!.id} />
      ) : (
        <>
          <div className="relative flex min-h-0 min-w-0 flex-1 flex-col">
          <ScrollArea
            ref={scrollRef}
            onScroll={onScroll}
            className="min-h-0 min-w-0 flex-1 overflow-y-auto overscroll-contain px-3 py-4 sm:px-5 sm:py-5"
          >
            <div className="mx-auto flex w-full max-w-3xl min-w-0 flex-col gap-4 sm:gap-5">
              <div className="flex min-h-6 items-center justify-center py-1">
                {loadingOlder && (
                  <p className="text-xs text-muted-foreground">Memuat pesan lama…</p>
                )}
                {!loadingOlder && hasMore && (
                  <button
                    type="button"
                    className="text-xs text-muted-foreground underline-offset-2 hover:underline"
                    onClick={() => {
                      const el = scrollRef.current
                      if (el) prevScrollHeightRef.current = el.scrollHeight
                      onLoadOlder?.()
                    }}
                  >
                    Muat 20 pesan sebelumnya
                  </button>
                )}
                {!loadingOlder && !hasMore && messages.length > 0 && (
                  <p className="text-[11px] text-muted-foreground/70">Awal percakapan</p>
                )}
              </div>

              {loading && messages.length === 0 && <ChatSkeleton />}
              {error && <p className="text-center text-sm text-red-400">{error}</p>}
              {!loading && !error && messages.length === 0 && (
                <p className="text-center text-sm text-muted-foreground">
                  Belum ada pesan di session ini.
                </p>
              )}

              {messages.map((m, i) => (
                <ChatBubble key={m.id || `${m.role}-${m.timestamp}-${i}`} message={m} known={mentionIndex.known} />
              ))}
              <div ref={bottomRef} />
            </div>
          </ScrollArea>
          {showJump && messages.length > 0 && (
            <button
              type="button"
              onClick={jumpToLatest}
              aria-label="Ke pesan terbaru"
              title="Ke pesan terbaru"
              className="absolute bottom-3 right-3 z-10 flex h-9 w-9 items-center justify-center rounded-full border border-border/70 bg-card/90 text-foreground shadow-lg backdrop-blur transition-colors hover:bg-accent sm:right-5"
            >
              <ArrowDown className="h-4 w-4" />
            </button>
          )}
          </div>

          <div className="shrink-0 border-t border-border/70 bg-background/60 px-3 py-2.5 backdrop-blur sm:px-5 sm:py-3">
            <ChatComposer
              sending={sending}
              cwd={session?.cwdMissing ? undefined : session?.cwd}
              disabled={!session}
              prefs={prefs}
              onPrefs={onPrefs}
              attach={attach}
              accept={acceptAttr(limits)}
              onSend={(t) => {
                if (session?.cwdMissing && onForkRequest) {
                  onForkRequest()
                  return
                }
                onSend?.(t, attach.attachments)
                attach.clear()
              }}
              onStop={() => onStop?.()}
            />
          </div>
        </>
      )}
      {errors.dialog}
    </div>
  )
}
