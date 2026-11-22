import { Bot, ChevronDown, LogOut, ShieldCheck, UserRound } from 'lucide-react'
import { useState } from 'react'
import { PopoverMenu } from '@/components/ui/popover-menu'
import { cn } from '@/lib/utils'

type Props = {
  username: string
  onOpenProfile: () => void
  onOpenSkills: () => void
  onLogout: () => void
  profileActive?: boolean
  skillsActive?: boolean
  /** Compact = hanya avatar (header sempit). */
  compact?: boolean
}

/** Menu akun di header kanan: profil, skills & agents, keluar. */
export function AccountMenu({
  username,
  onOpenProfile,
  onOpenSkills,
  onLogout,
  profileActive = false,
  skillsActive = false,
  compact = false,
}: Props) {
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  const open = Boolean(anchor)
  const active = profileActive || skillsActive
  const menuItem = 'flex w-full items-center gap-2 rounded-lg px-2.5 py-2 text-left text-sm hover:bg-white/5'

  return (
    <>
      <button
        type="button"
        onClick={(e) => setAnchor((cur) => (cur ? null : e.currentTarget))}
        aria-haspopup="menu"
        aria-expanded={open}
        title="Menu akun"
        className={cn(
          'flex shrink-0 items-center gap-2 rounded-xl border border-transparent px-1.5 py-1 text-left transition-colors hover:bg-white/5',
          (open || active) && 'border-border/60 bg-white/5',
          compact ? 'h-9 w-9 justify-center px-0' : 'max-w-[11rem] sm:max-w-[14rem]',
        )}
      >
        <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-primary text-xs font-semibold uppercase text-primary-foreground">
          {username.slice(0, 1)}
        </span>
        {!compact && (
          <>
            <span className="min-w-0 flex-1 hidden sm:block">
              <span className="block truncate text-xs font-medium leading-tight" title={username}>
                {username}
              </span>
              <span className="block truncate text-[10px] leading-tight text-muted-foreground">
                {skillsActive ? 'Skills & Agents' : 'Profil & keamanan'}
              </span>
            </span>
            <ChevronDown className={cn('hidden h-3.5 w-3.5 shrink-0 text-muted-foreground sm:block', open && 'rotate-180')} />
          </>
        )}
      </button>
      {anchor && (
        <PopoverMenu anchor={anchor} onClose={() => setAnchor(null)} width={232}>
          <div className="flex items-center gap-2.5 px-2.5 py-2">
            <UserRound className="h-4 w-4 text-muted-foreground" />
            <p className="min-w-0 flex-1 truncate text-xs text-muted-foreground">Masuk sebagai {username}</p>
          </div>
          <div className="my-1 h-px bg-border/60" />
          <button
            type="button"
            className={menuItem}
            onClick={() => {
              setAnchor(null)
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
              setAnchor(null)
              onOpenSkills()
            }}
          >
            <Bot className="h-4 w-4 text-sky-400" />
            Skills & Agents
          </button>
          <button
            type="button"
            className={menuItem}
            onClick={() => {
              setAnchor(null)
              onLogout()
            }}
          >
            <LogOut className="h-4 w-4 text-red-400" />
            Keluar
          </button>
        </PopoverMenu>
      )}
    </>
  )
}
