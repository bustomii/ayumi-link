import {
  ChevronDown,
  ChevronRight,
  ChevronUp,
  EyeOff,
  Folder,
  FolderInput,
  LogOut,
  ShieldCheck,
  MessageSquare,
  MoreHorizontal,
  Pencil,
  Plus,
  RefreshCw,
  Trash2,
  UserRound,
  X,
} from 'lucide-react'
import { useCallback, useEffect, useMemo, useState } from 'react'
import type { Session } from '@/api/sessions'
import { Button } from '@/components/ui/button'
import { SidebarSkeleton } from '@/components/sessions/SidebarSkeleton'
import { PopoverMenu } from '@/components/ui/popover-menu'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Skeleton } from '@/components/ui/skeleton'
import { groupSessionsByFolder } from '@/lib/session-groups'
import { cn } from '@/lib/utils'

type Props = {
  sessions: Session[]
  activeId: string | null
  loading: boolean
  open: boolean
  onClose: () => void
  onSelect: (id: string) => void
  onRefresh: () => void
  onNewSession: () => void
  onMoveRequest: (s: Session) => void
  onDeleteRequest: (s: Session) => void
  onRenameSession: (s: Session) => void
  onRenameGroup: (name: string) => void
  username: string
  onLogout: () => void
  onOpenProfile: () => void
  profileActive?: boolean
}

const COLLAPSE_KEY = 'ayumi-link:collapsed-folders-v3'

function loadCollapsed(): Record<string, boolean> {
  try {
    const raw = localStorage.getItem(COLLAPSE_KEY)
    if (!raw) return {}
    return JSON.parse(raw) as Record<string, boolean>
  } catch {
    return {}
  }
}

function saveCollapsed(map: Record<string, boolean>) {
  try {
    localStorage.setItem(COLLAPSE_KEY, JSON.stringify(map))
  } catch {
    /* ignore */
  }
}

function formatTime(iso: string) {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  return d.toLocaleString('id-ID', {
    day: '2-digit',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function initial(title: string) {
  const t = title.trim()
  return t ? t[0]!.toUpperCase() : '?'
}

export function SessionSidebar({
  sessions,
  activeId,
  loading,
  open,
  onClose,
  onSelect,
  onRefresh,
  onNewSession,
  onMoveRequest,
  onDeleteRequest,
  onRenameSession,
  onRenameGroup,
  username,
  onLogout,
  onOpenProfile,
  profileActive = false,
}: Props) {
  const groups = useMemo(() => groupSessionsByFolder(sessions), [sessions])
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({})
  const [ready, setReady] = useState(false)
  const [menu, setMenu] = useState<{ key: string; anchor: HTMLElement } | null>(null)
  const closeMenu = useCallback(() => setMenu(null), [])
  const toggleMenu = (key: string, anchor: HTMLElement) =>
    setMenu((m) => (m?.key === key ? null : { key, anchor }))

  useEffect(() => {
    const saved = loadCollapsed()
    const next: Record<string, boolean> = {}
    for (const g of groups) {
      next[g.key] = saved[g.key] !== undefined ? saved[g.key]! : false
    }
    if (activeId) {
      const g = groups.find((x) => x.sessions.some((s) => s.id === activeId))
      if (g) next[g.key] = false
    }
    setCollapsed(next)
    setReady(true)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [groups.length])

  useEffect(() => {
    if (!ready || !activeId) return
    const group = groups.find((g) => g.sessions.some((s) => s.id === activeId))
    if (!group) return
    setCollapsed((prev) => {
      if (prev[group.key] === false) return prev
      const next = { ...prev, [group.key]: false }
      saveCollapsed(next)
      return next
    })
  }, [activeId, groups, ready])

  function toggleGroup(key: string) {
    setCollapsed((prev) => {
      const next = { ...prev, [key]: !prev[key] }
      saveCollapsed(next)
      return next
    })
  }

  const menuItem = 'flex w-full items-center gap-2 rounded-lg px-2.5 py-2 text-left text-sm hover:bg-white/5'

  function renderMenu(m: { key: string; anchor: HTMLElement }) {
    if (m.key === 'user') {
      return (
        <PopoverMenu anchor={m.anchor} onClose={closeMenu} width={232}>
          <div className="flex items-center gap-2.5 px-2.5 py-2">
            <UserRound className="h-4 w-4 text-muted-foreground" />
            <p className="min-w-0 flex-1 truncate text-xs text-muted-foreground">Masuk sebagai {username}</p>
          </div>
          <div className="my-1 h-px bg-border/60" />
          <button
            type="button"
            className={menuItem}
            onClick={() => {
              closeMenu()
              onOpenProfile()
            }}
          >
            <ShieldCheck className="h-4 w-4 text-emerald-400" />
            Profil & keamanan
          </button>
          <button
            type="button"
            className={menuItem}
            onClick={() => {
              closeMenu()
              onLogout()
            }}
          >
            <LogOut className="h-4 w-4 text-red-400" />
            Keluar
          </button>
        </PopoverMenu>
      )
    }
    if (m.key.startsWith('group:')) {
      const group = groups.find((g) => `group:${g.key}` === m.key)
      if (!group) return null
      return (
        <PopoverMenu anchor={m.anchor} onClose={closeMenu} width={192}>
          <button
            type="button"
            className={menuItem}
            onClick={() => {
              closeMenu()
              onRenameGroup(group.label)
            }}
          >
            <Pencil className="h-4 w-4 text-sky-400" />
            Ubah nama folder
          </button>
        </PopoverMenu>
      )
    }
    const target = sessions.find((x) => x.id === m.key)
    if (!target) return null
    return (
      <PopoverMenu anchor={m.anchor} onClose={closeMenu} width={208}>
        <button
          type="button"
          className={menuItem}
          onClick={() => {
            closeMenu()
            onRenameSession(target)
          }}
        >
          <Pencil className="h-4 w-4 text-emerald-400" />
          Ubah nama
        </button>
        <button
          type="button"
          className={menuItem}
          onClick={() => {
            closeMenu()
            onMoveRequest(target)
          }}
        >
          <FolderInput className="h-4 w-4 text-sky-400" />
          Pindah ke folder
        </button>
        <button
          type="button"
          className={menuItem}
          onClick={() => {
            closeMenu()
            onDeleteRequest(target)
          }}
        >
          {target.local ? (
            <Trash2 className="h-4 w-4 text-red-400" />
          ) : (
            <EyeOff className="h-4 w-4 text-amber-400" />
          )}
          {target.local ? 'Hapus permanen' : 'Sembunyikan'}
        </button>
      </PopoverMenu>
    )
  }

  return (
    <>
      <div
        className={cn(
          'fixed inset-0 z-40 bg-black/55 backdrop-blur-[2px] transition-opacity md:hidden',
          open ? 'opacity-100' : 'pointer-events-none opacity-0',
        )}
        onClick={onClose}
        aria-hidden={!open}
      />

      <aside
        className={cn(
          // Mobile: drawer fixed. Desktop: panel shrink; bisa di-hide (bukan selalu md:translate-x-0).
          'fixed left-0 z-50 flex w-[min(100%,19rem)] flex-col border-r border-border/80 bg-sidebar transition-transform duration-200 md:static md:z-auto md:h-full md:shrink-0 top-[var(--app-top)] h-[var(--app-height)] max-h-[var(--app-height)] pt-[var(--safe-t)] pb-[var(--safe-b)] pl-[var(--safe-l)] md:top-auto md:h-full md:max-h-none md:pt-0 md:pb-0 md:pl-0',
          open ? 'translate-x-0 md:w-80' : '-translate-x-full md:pointer-events-none md:w-0 md:translate-x-0 md:overflow-hidden md:border-0 md:p-0',
        )}
        aria-hidden={!open}
      >
        <div className="flex items-center justify-between gap-2 border-b border-border/80 bg-gradient-to-b from-white/[0.03] to-transparent px-3 py-3.5">
          <div className="min-w-0">
            <p className="text-sm font-semibold tracking-tight">Sessions</p>
            {loading && sessions.length === 0 ? (
              <Skeleton className="mt-1 h-3 w-24" />
            ) : (
              <p className="text-[11px] text-muted-foreground">
                {groups.length} folder · {sessions.length} chat
              </p>
            )}
          </div>
          <div className="flex items-center gap-1">
            <Button
              variant="ghost"
              size="icon"
              onClick={onNewSession}
              title="Session baru"
              className="rounded-full"
            >
              <Plus className="h-4 w-4" />
            </Button>
            <Button
              variant="ghost"
              size="icon"
              onClick={onRefresh}
              disabled={loading}
              title="Refresh"
              className="rounded-full"
            >
              <RefreshCw className={cn('h-4 w-4', loading && 'animate-spin')} />
            </Button>
            <Button
              variant="ghost"
              size="icon"
              className="rounded-full"
              onClick={onClose}
              aria-label="Sembunyikan sessions"
              title="Sembunyikan sidebar"
            >
              <X className="h-4 w-4" />
            </Button>
          </div>
        </div>

        <ScrollArea className="min-h-0 flex-1 overflow-y-auto">
          <div className="space-y-3 p-2.5">
            {loading && sessions.length === 0 && <SidebarSkeleton />}
            {sessions.length === 0 && !loading && (
              <p className="px-2 py-8 text-center text-xs text-muted-foreground">
                Tidak ada session di ~/.claude
              </p>
            )}

            {groups.map((group) => {
              const isCollapsed = Boolean(collapsed[group.key])
              const hasActive = group.sessions.some((s) => s.id === activeId)

              return (
                <section key={group.key} className="min-w-0">
                  {/* Lapisan dasar solid: header yang menempel tidak boleh tembus pandang */}
                  <div className="sticky top-0 z-10 -mx-1 bg-sidebar px-1 pb-1.5">
                    <div className="relative">
                      <button
                        type="button"
                        onClick={() => toggleGroup(group.key)}
                        title={group.path || group.label}
                        className={cn(
                          'flex w-full cursor-pointer items-center gap-2 rounded-lg border py-2 pl-2.5 pr-9 text-left transition-colors',
                          hasActive
                            ? 'border-accent/30 bg-accent/10'
                            : 'border-border/60 bg-muted/40 hover:bg-muted/70',
                        )}
                      >
                        {isCollapsed ? (
                          <ChevronRight className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
                        ) : (
                          <ChevronDown className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
                        )}
                        <Folder
                          className={cn(
                            'h-3.5 w-3.5 shrink-0',
                            hasActive ? 'text-sky-300' : 'text-amber-400/90',
                          )}
                        />
                        <span className="min-w-0 flex-1 truncate text-xs font-semibold tracking-tight">
                          {group.label}
                        </span>
                        <span className="shrink-0 rounded-full bg-black/30 px-1.5 py-0.5 text-[10px] tabular-nums text-muted-foreground">
                          {group.sessions.length}
                        </span>
                        {group.liveCount > 0 && (
                          <span
                            className="h-1.5 w-1.5 shrink-0 rounded-full bg-emerald-400"
                            title="Ada session live"
                          />
                        )}
                      </button>
                      <button
                        type="button"
                        aria-label="Aksi folder"
                        title="Aksi folder"
                        onClick={(e) => toggleMenu(`group:${group.key}`, e.currentTarget)}
                        className="absolute right-1.5 top-1/2 flex h-6 w-6 -translate-y-1/2 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-white/10 hover:text-foreground"
                      >
                        <MoreHorizontal className="h-4 w-4" />
                      </button>
                    </div>
                  </div>

                  {!isCollapsed && (
                    <div className="space-y-1.5 pl-1">
                      {group.sessions.map((s) => {
                        const active = activeId === s.id
                        return (
                          <div key={s.id} className="relative">
                          <button
                            type="button"
                            aria-current={active ? 'page' : undefined}
                            onClick={() => {
                              onSelect(s.id)
                              onClose()
                            }}
                            className={cn(
                              'group relative flex w-full min-w-0 cursor-pointer items-start gap-3 rounded-xl border px-3 py-2.5 text-left transition-all duration-150',
                              'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/70',
                              'active:scale-[0.98]',
                              active
                                ? 'border-accent/40 bg-accent/15 shadow-[inset_3px_0_0_0_#3d8bfd] ring-1 ring-accent/25'
                                : 'border-transparent bg-transparent hover:border-border hover:bg-muted/70 hover:shadow-sm',
                            )}
                          >
                            <div
                              className={cn(
                                'mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg text-[11px] font-semibold transition-colors',
                                active
                                  ? 'bg-accent text-accent-foreground shadow-sm'
                                  : 'bg-muted text-muted-foreground group-hover:bg-muted-foreground/15 group-hover:text-foreground',
                              )}
                            >
                              {initial(s.title)}
                            </div>

                            <div className="min-w-0 flex-1">
                              <div className="flex min-w-0 items-center gap-2 pr-10">
                                <span
                                  className={cn(
                                    'truncate text-sm font-medium',
                                    active ? 'text-foreground' : 'text-foreground/90',
                                  )}
                                >
                                  {s.title}
                                </span>
                                {s.live && (
                                  <span
                                    className="inline-flex shrink-0 items-center gap-1 rounded-full bg-emerald-500/15 px-1.5 py-0.5 text-[9px] font-semibold uppercase tracking-wide text-emerald-400"
                                    title="Live"
                                  >
                                    <span className="h-1.5 w-1.5 animate-pulse rounded-full bg-emerald-400" />
                                    live
                                  </span>
                                )}
                              </div>

                              <div className="mt-1 flex min-w-0 items-center gap-2">
                                <span className="inline-flex max-w-[60%] items-center gap-1 truncate rounded-md bg-black/25 px-1.5 py-0.5 text-[10px] text-muted-foreground">
                                  <MessageSquare className="h-3 w-3 shrink-0 opacity-70" />
                                  <span className="truncate">{s.project || 'project'}</span>
                                </span>
                                <span className="shrink-0 text-[10px] text-muted-foreground/80">
                                  {formatTime(s.updatedAt)}
                                </span>
                              </div>
                            </div>

                            {active && (
                              <span className="absolute right-2.5 top-2.5 rounded-full bg-accent/20 px-1.5 py-0.5 text-[9px] font-semibold uppercase tracking-wide text-sky-300">
                                aktif
                              </span>
                            )}
                          </button>
                          <button
                            type="button"
                            aria-label="Aksi session"
                            title="Aksi session"
                            onClick={(e) => toggleMenu(s.id, e.currentTarget)}
                            className="absolute right-2 bottom-2 flex h-6 w-6 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-white/10 hover:text-foreground"
                          >
                            <MoreHorizontal className="h-4 w-4" />
                          </button>
                          </div>
                        )
                      })}
                    </div>
                  )}
                </section>
              )
            })}
          </div>
        </ScrollArea>

        {menu && renderMenu(menu)}

        <div className="shrink-0 border-t border-border/80 p-2">
          <button
            type="button"
            onClick={(e) => toggleMenu('user', e.currentTarget)}
            aria-haspopup="menu"
            aria-expanded={menu?.key === 'user'}
            title="Menu akun"
            className={cn(
              'flex w-full items-center gap-2.5 rounded-xl px-2 py-2 text-left transition-colors hover:bg-white/5',
              (menu?.key === 'user' || profileActive) && 'bg-white/5',
            )}
          >
            <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-primary text-xs font-semibold uppercase text-primary-foreground">
              {username.slice(0, 1)}
            </span>
            <span className="min-w-0 flex-1">
              <span className="block truncate text-sm font-medium" title={username}>
                {username}
              </span>
              <span className="block text-[11px] text-muted-foreground">Profil & keamanan</span>
            </span>
            <ChevronUp className="h-4 w-4 shrink-0 text-muted-foreground" />
          </button>
        </div>
      </aside>
    </>
  )
}
