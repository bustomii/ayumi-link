import { Field, PasswordInput } from '@/components/auth/Field'
import { Input } from '@/components/ui/input'

type Props = {
  password: string
  code: string
  onPassword: (v: string) => void
  onCode: (v: string) => void
  disabled?: boolean
}

/** Verifikasi ulang sebelum aksi sensitif: password + kode 2FA (atau kode pemulihan). */
export function StepUpFields({ password, code, onPassword, onCode, disabled }: Props) {
  return (
    <>
      <Field label="Password saat ini">
        <PasswordInput
          value={password}
          onChange={(e) => onPassword(e.target.value)}
          autoComplete="current-password"
          disabled={disabled}
          required
        />
      </Field>
      <Field label="Kode 2FA" hint="6 digit dari aplikasi autentikator, atau salah satu kode pemulihan.">
        <Input
          value={code}
          onChange={(e) => onCode(e.target.value)}
          inputMode="text"
          autoComplete="one-time-code"
          placeholder="000000"
          className="font-mono tracking-widest"
          disabled={disabled}
          required
        />
      </Field>
    </>
  )
}
