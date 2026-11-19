import { AlertTriangle, ShieldCheck } from 'lucide-react'
import { useState } from 'react'
import type { ProfileData } from '@/api/profile'
import { ChangeAuthenticatorDialog } from '@/components/profile/ChangeAuthenticatorDialog'
import { RegenerateRecoveryDialog } from '@/components/profile/RegenerateRecoveryDialog'
import { InfoRow, Section } from '@/components/profile/Section'
import { Button } from '@/components/ui/button'
import { formatDate } from '@/lib/device'
import { cn } from '@/lib/utils'

type Props = { profile: ProfileData; onChanged: () => void }

/** Pengaturan autentikasi dua langkah (2FA TOTP): status, kode pemulihan, ganti autentikator. */
export function TwoFactorSection({ profile, onChanged }: Props) {
  const [dialog, setDialog] = useState<'recovery' | 'authenticator' | null>(null)
  const low = profile.recoveryRemaining <= 2

  return (
    <>
      <Section
        icon={ShieldCheck}
        tone="text-emerald-400"
        title="Autentikasi dua langkah (2FA)"
        description="Setiap login meminta password dan kode 6 digit dari aplikasi autentikator."
        aside={
          <span className="inline-flex shrink-0 items-center gap-1 rounded-full bg-emerald-500/15 px-2 py-0.5 text-[11px] font-medium text-emerald-400">
            <span className="h-1.5 w-1.5 rounded-full bg-emerald-400" />
            Aktif
          </span>
        }
      >
        <dl className="divide-y divide-border/50">
          <InfoRow label="Autentikator sejak">{formatDate(profile.twoFactorSince) || '—'}</InfoRow>
          <InfoRow label="Kode pemulihan">
            <span className={cn(low && 'text-amber-400')}>
              {profile.recoveryRemaining} dari 8 tersisa
            </span>
          </InfoRow>
        </dl>

        {low && (
          <p className="mt-3 flex items-start gap-2 rounded-lg bg-amber-500/10 px-3 py-2 text-xs text-amber-300">
            <AlertTriangle className="mt-px h-4 w-4 shrink-0" />
            {profile.recoveryRemaining === 0
              ? 'Kode pemulihan habis. Buat kode baru sekarang agar tidak terkunci bila HP hilang.'
              : 'Kode pemulihan hampir habis. Sebaiknya buat kode baru.'}
          </p>
        )}

        <div className="mt-4 flex flex-wrap gap-2">
          <Button variant="outline" onClick={() => setDialog('recovery')}>
            Buat ulang kode pemulihan
          </Button>
          <Button variant="outline" onClick={() => setDialog('authenticator')}>
            Ganti autentikator
          </Button>
        </div>
        <p className="mt-3 text-[11px] text-muted-foreground">
          2FA wajib dan tidak bisa dimatikan. Untuk perangkat baru atau HP yang hilang, gunakan “Ganti autentikator”.
        </p>
      </Section>

      {dialog === 'recovery' && <RegenerateRecoveryDialog onClose={() => setDialog(null)} onDone={onChanged} />}
      {dialog === 'authenticator' && <ChangeAuthenticatorDialog onClose={() => setDialog(null)} onDone={onChanged} />}
    </>
  )
}
