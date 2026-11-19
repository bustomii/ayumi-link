import { useCallback, useEffect, useMemo, useState } from 'react'
import { fetchSessionDetail, deleteSession, moveSession, renameGroup, renameSession } from '@/api/sessions'
import type { Attachment, Message, Session } from '@/api/types'
import { ChatPanel } from '@/components/chat/ChatPanel'
import { ProfilePage } from '@/components/profile/ProfilePage'
import { NewSessionDialog } from '@/components/sessions/NewSessionDialog'
import { DeleteSessionDialog, MoveSessionDialog, RenameDialog } from '@/components/sessions/SessionDialogs'
import { SessionSidebar } from '@/components/sessions/SessionSidebar'
import { useChats } from '@/hooks/useChats'
import { useErrorDialog } from '@/hooks/useErrorDialog'
import { useHashView } from '@/hooks/useHashView'
import { useLatest } from '@/hooks/useLatest'
import { PAGE_SIZE, useSessionDetail } from '@/hooks/useSessionDetail'
import { useSessionList } from '@/hooks/useSessionList'
import { useSessionRoute } from '@/hooks/useSessionRoute'
import { usePrefs } from '@/hooks/usePrefs'

type Props = { username: string; onLogout: () => void }

type Draft = { key: string; session: Session }

type RenameTarget = { kind: 'session'; session: Session } | { kind: 'group'; name: string }

function makeDraft(cwd: string): Draft {
  return {
    key: `draft:${Date.now()}`,
    session: {
      id: '',
      title: 'Session baru',
      cwd,
      project: cwd.split('/').filter(Boolean).pop() ?? cwd,
      updatedAt: new Date().toISOString(),
      live: false,
    },
  }
}

/** Tampilan utama setelah login: sidebar session + panel chat. */
export function Workspace({ username, onLogout }: Props) {
  const [activeId, go] = useSessionRoute()
  const [draft, setDraft] = useState<Draft | null>(null)
  const [sidebarOpen, setSidebarOpen] = useState(() => {
    try {
      const v = localStorage.getItem('ayumi-link:sidebar-open')
      if (v === '0') return false
      if (v === '1') return true
    } catch {
      /* ignore */
    }
    return typeof window !== 'undefined' && window.matchMedia('(min-width: 768px)').matches
  })

  useEffect(() => {
    try {
      localStorage.setItem('ayumi-link:sidebar-open', sidebarOpen ? '1' : '0')
    } catch {
      /* ignore */
    }
  }, [sidebarOpen])
  const [newDialog, setNewDialog] = useState(false)
  const [moveTarget, setMoveTarget] = useState<Session | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<Session | null>(null)
  const [deleting, setDeleting] = useState(false)
  const [renameTarget, setRenameTarget] = useState<RenameTarget | null>(null)
  const [renaming, setRenaming] = useState(false)
  const errors = useErrorDialog()
  const [prefs, setPrefs] = usePrefs()
  const [view, setView] = useHashView()

  const list = useSessionList()
  const sessionsRef = useLatest(list.sessions)
  const detail = useSessionDetail(activeId, (id) => sessionsRef.current.find((s) => s.id === id) ?? null)
  const chats = useChats((e) => errors.show(e, 'Mengirim pesan'))

  // Kunci tampilan saat ini: id session, atau kunci draft untuk session baru.
  const viewKey = activeId ?? draft?.key ?? ''
  const viewKeyRef = useLatest(viewKey)
  const pending = chats.chats[viewKey] ?? null

  const selectSession = useCallback(
    (id: string) => {
      setDraft(null)
      setView('chat')
      go(id)
    },
    [go, setView],
  )

  const startDraft = useCallback((cwd: string) => {
    setNewDialog(false)
    setDraft(makeDraft(cwd))
    setView('chat')
    go(null)
    setSidebarOpen(false)
  }, [go, setView])

  const sendMessage = useCallback(
    async (text: string, atts: Attachment[]) => {
      const key = viewKeyRef.current
      if (!key) return
      const isDraft = activeId === null && draft !== null
      const sid = await chats.send(
        key,
        {
          sessionId: isDraft ? undefined : key,
          cwd: isDraft ? draft.session.cwd : undefined,
          message: text,
          permissionMode: prefs.permissionMode,
          model: prefs.model,
          effort: prefs.effort,
          attachments: atts.map((a) => a.id),
        },
        atts,
      )
      // Muat ulang dari transcript hanya bila pengguna masih melihat session ini; kalau sudah
      // pindah, balasan tetap tersimpan dan terbuka saat session itu dipilih lagi.
      if (sid && viewKeyRef.current === key) {
        try {
          const data = await fetchSessionDetail(sid, { limit: PAGE_SIZE })
          if (viewKeyRef.current === key) {
            detail.adopt(sid, data)
            setDraft(null)
            go(sid)
          }
        } catch {
          /* transcript belum ada (mis. gagal sebelum mulai) */
        }
      }
      if (sid) void list.reload()
      chats.finish(key)
    },
    [activeId, chats, detail, draft, go, list, prefs, viewKeyRef],
  )

  const doMove = useCallback(
    async (group: string) => {
      if (!moveTarget) return
      try {
        await moveSession(moveTarget.id, group)
        void list.reload()
      } catch (e) {
        errors.show(e, 'Memindahkan session')
      }
      setMoveTarget(null)
    },
    [errors, list, moveTarget],
  )

  const doRename = useCallback(
    async (value: string) => {
      if (!renameTarget) return
      setRenaming(true)
      try {
        if (renameTarget.kind === 'session') await renameSession(renameTarget.session.id, value)
        else await renameGroup(renameTarget.name, value)
        await list.reload()
        setRenameTarget(null)
      } catch (e) {
        errors.show(e, 'Mengubah nama')
        setRenameTarget(null)
      } finally {
        setRenaming(false)
      }
    },
    [errors, list, renameTarget],
  )

  const doDelete = useCallback(async () => {
    if (!deleteTarget) return
    setDeleting(true)
    try {
      await deleteSession(deleteTarget.id)
      chats.stop(deleteTarget.id)
      if (activeId === deleteTarget.id) go(null, { replace: true })
      void list.reload()
    } catch (e) {
      errors.show(e, deleteTarget.local ? 'Menghapus session' : 'Menyembunyikan session')
    } finally {
      setDeleting(false)
      setDeleteTarget(null)
    }
  }, [activeId, chats, deleteTarget, errors, go, list])

  const displayMessages = useMemo<Message[]>(() => {
    if (!pending) return detail.messages
    // Transcript bisa sudah memuat pesan user ini (session dibuka saat balasan masih berjalan).
    const userAlready = detail.messages
      .slice(-3)
      .some((m) => m.role === 'user' && m.content.trim() === pending.user.trim())
    return [
      ...detail.messages,
      ...(userAlready
        ? []
        : [
            {
              id: 'pending-user',
              role: 'user',
              content: pending.user,
              timestamp: pending.startedAt,
              attachments: pending.attachments,
            },
          ]),
      {
        id: 'pending-assistant',
        role: 'assistant',
        content: pending.assistant || '…',
        tools: pending.tools,
        timestamp: pending.startedAt,
      },
    ]
  }, [detail.messages, pending])

  // Draft hanya tampil saat tidak ada session terbuka di URL.
  const listed = activeId ? list.sessions.find((x) => x.id === activeId) : undefined
  const shownSession =
    activeId === null
      ? (draft?.session ?? null)
      : detail.session && listed
        ? { ...detail.session, title: listed.title } // judul ikut berubah setelah rename
        : detail.session

  // Gagal memuat daftar session (mis. backend mati) ditampilkan sebagai popup dengan tombol coba lagi.
  const listError = list.error
  const showError = errors.show
  const reloadList = list.reload
  useEffect(() => {
    if (listError) showError(listError, 'Memuat daftar session', () => void reloadList())
  }, [listError, showError, reloadList])

  const shownTitle = shownSession?.title
  useEffect(() => {
    if (view === 'profile') document.title = 'Profil & keamanan · Ayumi Link'
    else document.title = shownTitle ? `${shownTitle} · Ayumi Link` : 'Ayumi Link'
  }, [shownTitle, view])

  const groupNames = useMemo(
    () => Array.from(new Set(list.sessions.map((s) => s.group).filter((g): g is string => Boolean(g)))),
    [list.sessions],
  )
  const recentCwds = useMemo(
    () => Array.from(new Set(list.sessions.map((s) => s.cwd).filter(Boolean))),
    [list.sessions],
  )

  return (
    <div className="app-shell flex min-h-0 overflow-hidden pt-[var(--safe-t)] pr-[var(--safe-r)] pb-[var(--safe-b)] pl-[var(--safe-l)]">
      <SessionSidebar
        sessions={list.sessions}
        activeId={activeId}
        loading={list.loading}
        open={sidebarOpen}
        username={username}
        onLogout={onLogout}
        onOpenProfile={() => {
          setView('profile')
          setSidebarOpen(false)
        }}
        profileActive={view === 'profile'}
        onClose={() => setSidebarOpen(false)}
        onSelect={selectSession}
        onNewSession={() => setNewDialog(true)}
        onMoveRequest={setMoveTarget}
        onDeleteRequest={setDeleteTarget}
        onRenameSession={(x) => setRenameTarget({ kind: 'session', session: x })}
        onRenameGroup={(name) => setRenameTarget({ kind: 'group', name })}
        onRefresh={() => void list.reload()}
      />

      <div className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
        {view === 'profile' ? (
          <ProfilePage
            prefs={prefs}
            onPrefs={setPrefs}
            onBack={() => setView('chat')}
            onOpenSidebar={() => setSidebarOpen((v) => !v)}
            onLogout={onLogout}
          />
        ) : (
          <ChatPanel
            session={shownSession}
            messages={displayMessages}
            loading={detail.loading || (list.loading && list.sessions.length === 0 && !draft)}
            loadingOlder={detail.loadingOlder}
            hasMore={detail.hasMore}
            error={detail.error}
            onNewSession={() => setNewDialog(true)}
            onOpenSidebar={() => setSidebarOpen((v) => !v)}
            onLoadOlder={() => void detail.loadOlder()}
            sending={pending !== null}
            onSend={(t, a) => void sendMessage(t, a)}
            onStop={() => chats.stop(viewKey)}
            prefs={prefs}
            onPrefs={setPrefs}
          />
        )}
      </div>

      {moveTarget && (
        <MoveSessionDialog
          session={moveTarget}
          groups={groupNames}
          onCancel={() => setMoveTarget(null)}
          onMove={(g) => void doMove(g)}
        />
      )}
      {renameTarget && (
        <RenameDialog
          key={renameTarget.kind === 'session' ? renameTarget.session.id : `group:${renameTarget.name}`}
          heading={renameTarget.kind === 'session' ? 'Ubah nama session' : 'Ubah nama folder'}
          description={
            renameTarget.kind === 'session'
              ? 'Nama baru hanya berlaku di Ayumi Link, tidak mengubah nama di Claude Desktop. Kosongkan untuk kembali ke nama asli.'
              : 'Nama baru hanya berlaku di Ayumi Link, tidak mengubah folder di Claude Desktop. Memakai nama folder yang sudah ada akan menggabungkan keduanya.'
          }
          initial={renameTarget.kind === 'session' ? renameTarget.session.title : renameTarget.name}
          maxLength={renameTarget.kind === 'session' ? 120 : 60}
          allowEmpty={renameTarget.kind === 'session'}
          busy={renaming}
          onCancel={() => setRenameTarget(null)}
          onSubmit={(v) => void doRename(v)}
        />
      )}
      {deleteTarget && (
        <DeleteSessionDialog
          session={deleteTarget}
          busy={deleting}
          onCancel={() => setDeleteTarget(null)}
          onConfirm={() => void doDelete()}
        />
      )}
      {errors.dialog}
      {newDialog && (
        <NewSessionDialog recentCwds={recentCwds} onCancel={() => setNewDialog(false)} onCreate={startDraft} />
      )}
    </div>
  )
}
