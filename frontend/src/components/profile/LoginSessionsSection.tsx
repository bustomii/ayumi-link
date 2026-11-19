import { Laptop, Loader2, LogOut, Smartphone, Trash2 } from 'lucide-react'
import { useState } from 'react'
import { revokeOtherSessions, revokeSession, type LoginSession, type ProfileData } from '@/api/profile'
import { Section } from '@/components/profile/Section'
import { Button } from '@/components/ui/button'
import { ConfirmDialog } from '@/components/ui/confirm-dialog'
import { describeUA, timeAgo } from '@/lib/device'

type Props = {
  profile: ProfileData
  onChanged: () => void
  onError: (e: unknown, action: string) => void
}

const isPhone = (ua: string) => /iPhone|iPad|iPod|Android/.test(ua)

function SessionRow({ s, busy, onRevoke }: { s: LoginSession; busy: boolean; onRevoke: () => void }) {
  const Icon = isPhone(s.ua) ? Smartphone : Laptop
  return (
    <li className="flex items-center gap-3 py-3">
      <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-background/60">
        <Icon className="h-4 w-4 text-muted-foreground" />
      </span>
      <div className="min-w-0 flex-1">
        <p className="flex flex-wrap items-center gap-x-2 text-sm font-medium">
          {describeUA(s.ua)}
          {s.current && (
            <span className="rounded-full bg-sky-500/15 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-sky-300">
              Perangkat ini
            </span>
          )}
        </p>
        <p className="truncate text-xs text-muted-foreground">
          {s.ip || 'IP tidak diketahui'} · aktif {timeAgo(s.lastSeen)} · masuk {timeAgo(s.createdAt)}
        </p>
      </div>
      {!s.current && (
        <Button variant="ghost" size="icon" onClick={onRevoke} disabled={busy} title="Keluarkan perangkat ini" aria-label="Keluarkan perangkat ini">
          {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : <Trash2 className="h-4 w-4 text-red-400" />}
        </Button>
      )}
    </li>
  )
}

/** Daftar perangkat yang sedang login, dengan opsi mengeluarkannya. */
export function LoginSessionsSection({ profile, onChanged, onError }: Props) {
  const [busyId, setBusyId] = useState<string | null>(null)
  const [confirmAll, setConfirmAll] = useState(false)
  const [busyAll, setBusyAll] = useState(false)
  const others = profile.sessions.filter((s) => !s.current).length

  async function revoke(id: string) {
    setBusyId(id)
    try {
      await revokeSession(id)
      onChanged()
    } catch (e) {
      onError(e, 'Mengeluarkan perangkat')
    } finally {
      setBusyId(null)
    }
  }

  async function revokeAll() {
    setBusyAll(true)
    try {
      await revokeOtherSessions()
      onChanged()
    } catch (e) {
      onError(e, 'Mengeluarkan perangkat lain')
    } finally {
      setBusyAll(false)
      setConfirmAll(false)
    }
  }

  return (
    <>
      <Section
        icon={Laptop}
        tone="text-sky-400"
        title="Perangkat yang sedang masuk"
        description="Keluarkan perangkat yang tidak dikenal atau sudah tidak dipakai."
      >
        <ul className="divide-y divide-border/50">
          {profile.sessions.map((s) => (
            <SessionRow key={s.id} s={s} busy={busyId === s.id} onRevoke={() => void revoke(s.id)} />
          ))}
        </ul>
        {others > 0 && (
          <Button variant="outline" className="mt-3" onClick={() => setConfirmAll(true)}>
            <LogOut className="h-4 w-4" />
            Keluarkan {others} perangkat lain
          </Button>
        )}
      </Section>

      {confirmAll && (
        <ConfirmDialog
          title="Keluarkan semua perangkat lain?"
          description={`${others} perangkat akan keluar dan harus login lagi dengan password dan 2FA. Perangkat ini tetap masuk.`}
          confirmLabel="Keluarkan"
          tone="danger"
          busy={busyAll}
          onConfirm={() => void revokeAll()}
          onCancel={() => setConfirmAll(false)}
        />
      )}
    </>
  )
}
