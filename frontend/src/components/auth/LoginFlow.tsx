import { ArrowLeft, Loader2 } from 'lucide-react'
import { useEffect, useState, type FormEvent } from 'react'
import { login, verifyLogin } from '@/api/auth'
import { ApiError } from '@/api/http'
import { AuthLayout } from '@/components/auth/AuthLayout'
import { CodeInput } from '@/components/auth/CodeInput'
import { Field, FormError, PasswordInput } from '@/components/auth/Field'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

function formatWait(totalSeconds: number) {
  const m = Math.floor(totalSeconds / 60)
  const s = totalSeconds % 60
  return `${m}:${String(s).padStart(2, '0')}`
}

/** Sisa detik hingga tenggat (berbasis waktu nyata, bukan hitungan interval). */
function useRemaining(deadline: number | null) {
  const [now, setNow] = useState(() => Date.now())
  const remaining = deadline === null ? 0 : Math.max(0, Math.ceil((deadline - now) / 1000))
  useEffect(() => {
    if (remaining <= 0) return
    const t = setInterval(() => setNow(Date.now()), 1000)
    return () => clearInterval(t)
  }, [remaining])
  return remaining
}

type Props = { onDone: () => void }

export function LoginFlow({ onDone }: Props) {
  const [step, setStep] = useState<'credentials' | 'code'>('credentials')
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [mfaToken, setMfaToken] = useState('')
  const [useRecovery, setUseRecovery] = useState(false)
  const [code, setCode] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [lockedUntil, setLockedUntil] = useState<number | null>(null)
  const wait = useRemaining(lockedUntil)
  const locked = wait > 0

  function fail(e: unknown) {
    if (e instanceof ApiError) {
      if (e.code === 'locked') {
        setLockedUntil(Date.now() + (e.retryAfter ?? 900) * 1000)
        setError(null)
        return
      }
      if (e.code === 'challenge_expired') {
        setStep('credentials')
        setCode('')
      }
      setError(e.message)
      return
    }
    setError('Tidak bisa menghubungi server')
  }

  async function submitCredentials(e: FormEvent) {
    e.preventDefault()
    if (busy || locked) return
    setBusy(true)
    setError(null)
    try {
      const res = await login({ username, password })
      setMfaToken(res.mfaToken)
      setPassword('')
      setCode('')
      setUseRecovery(false)
      setStep('code')
    } catch (err) {
      fail(err)
    } finally {
      setBusy(false)
    }
  }

  async function submitCode(value: string) {
    const c = value.trim()
    if (!c || busy || locked) return
    setBusy(true)
    setError(null)
    try {
      await verifyLogin({ mfaToken, code: c })
      onDone()
    } catch (err) {
      setCode('')
      fail(err)
    } finally {
      setBusy(false)
    }
  }

  const lockNotice = locked && (
    <FormError>Terlalu banyak percobaan. Coba lagi dalam {formatWait(wait)}.</FormError>
  )

  if (step === 'credentials') {
    return (
      <AuthLayout title="Masuk" description="Gunakan akun Ayumi Link Anda.">
        <form onSubmit={submitCredentials} className="space-y-4">
          <Field label="Username">
            <Input
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              autoComplete="username"
              autoCapitalize="none"
              autoFocus
              required
            />
          </Field>
          <Field label="Password">
            <PasswordInput
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              autoComplete="current-password"
              required
            />
          </Field>
          {lockNotice}
          {error && <FormError>{error}</FormError>}
          <Button type="submit" className="w-full" disabled={busy || locked || !username || !password}>
            {busy && <Loader2 className="h-4 w-4 animate-spin" />}
            Lanjut
          </Button>
        </form>
      </AuthLayout>
    )
  }

  return (
    <AuthLayout
      title="Verifikasi dua langkah"
      description={
        useRecovery
          ? 'Masukkan salah satu kode pemulihan yang Anda simpan saat setup.'
          : 'Masukkan 6 digit kode dari aplikasi autentikator Anda.'
      }
    >
      <div className="space-y-4">
        {useRecovery ? (
          <form
            className="space-y-4"
            onSubmit={(e) => {
              e.preventDefault()
              void submitCode(code)
            }}
          >
            <Input
              value={code}
              onChange={(e) => setCode(e.target.value.toUpperCase())}
              placeholder="XXXXX-XXXXX"
              autoComplete="off"
              autoCapitalize="characters"
              autoFocus
              aria-label="Kode pemulihan"
              className="py-3 text-center font-mono tracking-widest"
            />
            <Button type="submit" className="w-full" disabled={busy || locked || code.trim().length < 10}>
              {busy && <Loader2 className="h-4 w-4 animate-spin" />}
              Verifikasi
            </Button>
          </form>
        ) : (
          <CodeInput value={code} onChange={setCode} onComplete={(c) => void submitCode(c)} disabled={busy || locked} />
        )}

        {lockNotice}
        {error && <FormError>{error}</FormError>}

        <div className="flex items-center justify-between text-xs">
          <button
            type="button"
            onClick={() => {
              setStep('credentials')
              setError(null)
              setCode('')
            }}
            className="inline-flex items-center gap-1 text-muted-foreground hover:text-foreground"
          >
            <ArrowLeft className="h-3.5 w-3.5" />
            Kembali
          </button>
          <button
            type="button"
            onClick={() => {
              setUseRecovery((v) => !v)
              setCode('')
              setError(null)
            }}
            className="text-primary hover:underline"
          >
            {useRecovery ? 'Pakai kode autentikator' : 'Pakai kode pemulihan'}
          </button>
        </div>
      </div>
    </AuthLayout>
  )
}
