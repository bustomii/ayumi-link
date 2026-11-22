import { ArrowLeft, KeyRound, LogOut, Menu, UserRound } from 'lucide-react'
import { useEffect, useState } from 'react'
import { AccountMenu } from '@/components/AccountMenu'
import { ClaudeAuthSection } from '@/components/profile/ClaudeAuthSection'
import { LoginSessionsSection } from '@/components/profile/LoginSessionsSection'
import { ChangePasswordDialog } from '@/components/profile/ChangePasswordDialog'
import { PrefsSection } from '@/components/profile/PrefsSection'
import { Section } from '@/components/profile/Section'
import { ServerSection } from '@/components/profile/ServerSection'
import { TwoFactorSection } from '@/components/profile/TwoFactorSection'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Skeleton } from '@/components/ui/skeleton'
import { useErrorDialog } from '@/hooks/useErrorDialog'
import { useProfile } from '@/hooks/useProfile'
import { formatDate } from '@/lib/device'
import type { Prefs } from '@/lib/chat-options'

type Props = {
  prefs: Prefs
  onPrefs: (p: Prefs) => void
  onBack: () => void
  onOpenSidebar: () => void
  onLogout: () => void
  username: string
  onOpenSkills: () => void
}

function ProfileSkeleton() {
  return (
    <div className="space-y-4" role="status" aria-label="Memuat profil">
      {[3, 4, 3, 2].map((rows, i) => (
        <div key={i} className="rounded-2xl border border-border/70 bg-card/70 p-5">
          <div className="flex items-center gap-3">
            <Skeleton className="h-9 w-9 rounded-xl" />
            <div className="space-y-2">
              <Skeleton className="h-3.5 w-40" />
              <Skeleton className="h-3 w-56 max-w-[50vw]" />
            </div>
          </div>
          <div className="mt-5 space-y-3">
            {Array.from({ length: rows }).map((_, j) => (
              <Skeleton key={j} className="h-4 w-full" />
            ))}
          </div>
        </div>
      ))}
    </div>
  )
}

/** Halaman profil & keamanan: akun, 2FA, perangkat yang masuk, pengaturan chat, info server. */
export function ProfilePage({ prefs, onPrefs, onBack, onOpenSidebar, onLogout, username, onOpenSkills }: Props) {
  const { data, loading, error, reload } = useProfile()
  const errors = useErrorDialog()
  const [passwordDialog, setPasswordDialog] = useState(false)

  const showError = errors.show
  // Gagal memuat profil: popup dengan tombol coba lagi.
  useEffect(() => {
    if (error && !data) showError(error, 'Memuat profil', () => void reload())
  }, [error, data, showError, reload])

  return (
    <div className="flex h-full min-h-0 min-w-0 flex-1 flex-col overflow-hidden bg-[radial-gradient(ellipse_at_top,_#151821_0%,_#0c0d10_50%)]">
      <header className="flex shrink-0 items-center gap-2 border-b border-border/70 bg-background/70 px-3 py-2.5 backdrop-blur sm:px-5 sm:py-3">
        <Button variant="ghost" size="icon" className="shrink-0" onClick={onOpenSidebar} aria-label="Buka sessions" title="Sessions">
          <Menu className="h-5 w-5" />
        </Button>
        <div className="min-w-0 flex-1">
          <h1 className="truncate text-sm font-semibold tracking-tight sm:text-base">Profil & keamanan</h1>
          <p className="truncate text-[11px] text-muted-foreground sm:text-xs">Akun, 2FA, dan perangkat yang masuk</p>
        </div>
        <Button variant="outline" size="sm" className="shrink-0" onClick={onBack}>
          <ArrowLeft className="h-4 w-4" />
          <span className="hidden sm:inline">Kembali ke chat</span>
        </Button>
        <AccountMenu
          username={username}
          onOpenProfile={() => {}}
          onOpenSkills={onOpenSkills}
          onLogout={onLogout}
          profileActive
        />
      </header>

      <ScrollArea className="min-h-0 flex-1 overflow-y-auto overscroll-contain px-3 py-4 sm:px-5 sm:py-6">
        <div className="mx-auto w-full max-w-2xl space-y-4">
          {loading && !data && <ProfileSkeleton />}

          {data && (
            <>
              <Section icon={UserRound} tone="text-sky-400" title="Akun" description="Satu akun pemilik untuk aplikasi ini.">
                <div className="flex flex-wrap items-center gap-3">
                  <span className="flex h-12 w-12 shrink-0 items-center justify-center rounded-full bg-primary text-lg font-semibold uppercase text-primary-foreground">
                    {data.username.slice(0, 1)}
                  </span>
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-sm font-semibold">{data.username}</p>
                    <p className="text-xs text-muted-foreground">Akun dibuat {formatDate(data.createdAt)}</p>
                  </div>
                  <div className="flex gap-2">
                    <Button variant="outline" onClick={() => setPasswordDialog(true)}>
                      <KeyRound className="h-4 w-4" />
                      Ubah password
                    </Button>
                    <Button variant="outline" onClick={onLogout}>
                      <LogOut className="h-4 w-4" />
                      Keluar
                    </Button>
                  </div>
                </div>
              </Section>

              <ClaudeAuthSection onError={(e, a) => showError(e, a)} />
              <TwoFactorSection profile={data} onChanged={() => void reload()} />
              <LoginSessionsSection profile={data} onChanged={() => void reload()} onError={(e, a) => showError(e, a)} />
              <PrefsSection prefs={prefs} onPrefs={onPrefs} />
              <ServerSection config={data.config} />
            </>
          )}
        </div>
      </ScrollArea>

      {passwordDialog && <ChangePasswordDialog onClose={() => { setPasswordDialog(false); void reload() }} />}
      {errors.dialog}
    </div>
  )
}
