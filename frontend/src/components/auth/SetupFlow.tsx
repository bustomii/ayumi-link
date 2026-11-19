import { Check, Copy, Loader2 } from 'lucide-react'
import { useState, type FormEvent } from 'react'
import { beginSetup, confirmSetup, type SetupInfo } from '@/api/auth'
import { ApiError } from '@/api/http'
import { AuthLayout } from '@/components/auth/AuthLayout'
import { CodeInput } from '@/components/auth/CodeInput'
import { RecoveryCodes } from '@/components/auth/RecoveryCodes'
import { Field, FormError, PasswordInput } from '@/components/auth/Field'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { copyText } from '@/lib/clipboard'

const MIN_PASSWORD = 12

function errMsg(e: unknown) {
  return e instanceof ApiError || e instanceof Error ? e.message : 'Terjadi kesalahan'
}

type Props = { onDone: () => void }

/** Setup pertama: akun → pindai QR / kunci 2FA → simpan kode pemulihan. */
export function SetupFlow({ onDone }: Props) {
  const [step, setStep] = useState<'form' | 'enroll' | 'recovery'>('form')
  const [setupToken, setSetupToken] = useState('')
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [info, setInfo] = useState<SetupInfo | null>(null)
  const [code, setCode] = useState('')
  const [recovery, setRecovery] = useState<string[]>([])
  const [copied, setCopied] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const mismatch = confirm.length > 0 && confirm !== password
  const formValid = setupToken && /^[a-z0-9._-]{3,32}$/.test(username) && password.length >= MIN_PASSWORD && !mismatch && confirm

  async function submitForm(e: FormEvent) {
    e.preventDefault()
    if (busy || !formValid) return
    setBusy(true)
    setError(null)
    try {
      setInfo(await beginSetup({ setupToken: setupToken.trim(), username, password }))
      setPassword('')
      setConfirm('')
      setStep('enroll')
    } catch (err) {
      setError(errMsg(err))
    } finally {
      setBusy(false)
    }
  }

  async function submitCode(value: string) {
    if (busy) return
    setBusy(true)
    setError(null)
    try {
      const res = await confirmSetup({ setupToken: setupToken.trim(), code: value })
      setRecovery(res.recoveryCodes)
      setStep('recovery')
    } catch (err) {
      setCode('')
      setError(errMsg(err))
    } finally {
      setBusy(false)
    }
  }

  function flash() {
    setCopied(true)
    setTimeout(() => setCopied(false), 1500)
  }

  if (step === 'form') {
    return (
      <AuthLayout title="Buat akun" description="Pertama kali: buat akun pemilik. 2FA diaktifkan di langkah berikutnya.">
        <form onSubmit={submitForm} className="space-y-4">
          <Field label="Token setup" hint="Lihat di log server (terminal tempat backend berjalan).">
            <Input
              value={setupToken}
              onChange={(e) => setSetupToken(e.target.value)}
              autoComplete="off"
              spellCheck={false}
              autoFocus
              required
            />
          </Field>
          <Field label="Username" hint="3-32 karakter: huruf kecil, angka, titik, strip, garis bawah.">
            <Input
              value={username}
              onChange={(e) => setUsername(e.target.value.toLowerCase())}
              autoComplete="username"
              autoCapitalize="none"
              required
            />
          </Field>
          <Field label="Password" hint={`Minimal ${MIN_PASSWORD} karakter`}>
            <PasswordInput value={password} onChange={(e) => setPassword(e.target.value)} autoComplete="new-password" required />
          </Field>
          <Field label="Ulangi password">
            <PasswordInput value={confirm} onChange={(e) => setConfirm(e.target.value)} autoComplete="new-password" required />
          </Field>
          {mismatch && <FormError>Password tidak sama.</FormError>}
          {error && <FormError>{error}</FormError>}
          <Button type="submit" className="w-full" disabled={busy || !formValid}>
            {busy && <Loader2 className="h-4 w-4 animate-spin" />}
            Lanjut
          </Button>
        </form>
      </AuthLayout>
    )
  }

  if (step === 'enroll' && info) {
    const grouped = info.secret.replace(/(.{4})/g, '$1 ').trim()
    return (
      <AuthLayout
        title="Aktifkan 2FA"
        description="Pindai QR dengan aplikasi autentikator (Google Authenticator, 1Password, Authy), lalu masukkan kodenya."
      >
        <div className="space-y-4">
          <div className="mx-auto w-fit rounded-xl bg-white p-2">
            <img src={info.qr} alt="QR code 2FA" width={200} height={200} className="block h-[200px] w-[200px]" />
          </div>
          <div className="rounded-lg bg-background/60 p-3">
            <p className="text-[11px] text-muted-foreground">Tidak bisa memindai? Masukkan kunci ini manual:</p>
            <div className="mt-1 flex items-center gap-2">
              <code className="min-w-0 flex-1 break-all font-mono text-xs">{grouped}</code>
              <Button
                type="button"
                variant="ghost"
                size="icon"
                aria-label="Salin kunci"
                onClick={async () => (await copyText(info.secret)) && flash()}
              >
                {copied ? <Check className="h-4 w-4 text-emerald-400" /> : <Copy className="h-4 w-4" />}
              </Button>
            </div>
          </div>
          <CodeInput value={code} onChange={setCode} onComplete={(c) => void submitCode(c)} disabled={busy} />
          {error && <FormError>{error}</FormError>}
          {busy && (
            <p className="flex items-center justify-center gap-2 text-xs text-muted-foreground">
              <Loader2 className="h-3.5 w-3.5 animate-spin" /> Memverifikasi…
            </p>
          )}
        </div>
      </AuthLayout>
    )
  }

  return (
    <AuthLayout
      title="Simpan kode pemulihan"
      description="Kode ini satu-satunya cara masuk bila HP autentikator hilang. Tiap kode hanya bisa dipakai sekali dan tidak akan ditampilkan lagi."
    >
      <RecoveryCodes codes={recovery} doneLabel="Mulai pakai Ayumi Link" onDone={onDone} />
    </AuthLayout>
  )
}
