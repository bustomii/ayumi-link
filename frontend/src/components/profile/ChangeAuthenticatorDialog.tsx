import { Check, Copy, Loader2, Smartphone } from 'lucide-react'
import { useState, type FormEvent } from 'react'
import type { SetupInfo } from '@/api/auth'
import { beginTwoFactorChange, confirmTwoFactorChange } from '@/api/profile'
import { CodeInput } from '@/components/auth/CodeInput'
import { FormError } from '@/components/auth/Field'
import { RecoveryCodes } from '@/components/auth/RecoveryCodes'
import { formErrorText } from '@/components/profile/form-errors'
import { StepUpFields } from '@/components/profile/StepUpFields'
import { Button } from '@/components/ui/button'
import { Modal } from '@/components/ui/modal'
import { copyText } from '@/lib/clipboard'

type Props = { onClose: () => void; onDone: () => void }

/** Ganti perangkat autentikator: verifikasi ulang → pindai QR baru → kode pemulihan baru. */
export function ChangeAuthenticatorDialog({ onClose, onDone }: Props) {
  const [step, setStep] = useState<'verify' | 'enroll' | 'recovery'>('verify')
  const [password, setPassword] = useState('')
  const [code, setCode] = useState('')
  const [info, setInfo] = useState<SetupInfo | null>(null)
  const [newCode, setNewCode] = useState('')
  const [recovery, setRecovery] = useState<string[]>([])
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)

  async function begin(e: FormEvent) {
    e.preventDefault()
    if (busy) return
    setBusy(true)
    setError(null)
    try {
      setInfo(await beginTwoFactorChange({ password, code: code.trim() }))
      setPassword('')
      setCode('')
      setStep('enroll')
    } catch (err) {
      setCode('')
      setError(formErrorText(err))
    } finally {
      setBusy(false)
    }
  }

  async function confirm(value: string) {
    if (busy) return
    setBusy(true)
    setError(null)
    try {
      const res = await confirmTwoFactorChange({ code: value })
      setRecovery(res.recoveryCodes)
      setStep('recovery')
      onDone()
    } catch (err) {
      setNewCode('')
      setError(formErrorText(err))
    } finally {
      setBusy(false)
    }
  }

  if (step === 'recovery') {
    return (
      <Modal label="Autentikator diganti" onClose={() => {}}>
        <p className="text-base font-semibold">Autentikator berhasil diganti</p>
        <p className="mb-4 mt-1 text-xs text-muted-foreground">
          Simpan kode pemulihan baru ini; kode lama sudah hangus dan tidak akan ditampilkan lagi.
        </p>
        <RecoveryCodes codes={recovery} doneLabel="Selesai" onDone={onClose} />
      </Modal>
    )
  }

  if (step === 'enroll' && info) {
    const grouped = info.secret.replace(/(.{4})/g, '$1 ').trim()
    return (
      <Modal label="Pindai autentikator baru" onClose={onClose}>
        <div className="space-y-4">
          <div>
            <p className="text-base font-semibold">Pindai di perangkat baru</p>
            <p className="mt-1 text-xs text-muted-foreground">
              Pindai QR dengan aplikasi autentikator, lalu masukkan kode 6 digit yang muncul. Autentikator lama
              masih berlaku sampai langkah ini selesai.
            </p>
          </div>
          <div className="mx-auto w-fit rounded-xl bg-white p-2">
            <img src={info.qr} alt="QR code 2FA baru" width={200} height={200} className="block h-[200px] w-[200px]" />
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
                onClick={async () => {
                  if (await copyText(info.secret)) {
                    setCopied(true)
                    setTimeout(() => setCopied(false), 1500)
                  }
                }}
              >
                {copied ? <Check className="h-4 w-4 text-emerald-400" /> : <Copy className="h-4 w-4" />}
              </Button>
            </div>
          </div>
          <CodeInput value={newCode} onChange={setNewCode} onComplete={(c) => void confirm(c)} disabled={busy} />
          {error && <FormError>{error}</FormError>}
          {busy && (
            <p className="flex items-center justify-center gap-2 text-xs text-muted-foreground">
              <Loader2 className="h-3.5 w-3.5 animate-spin" /> Memverifikasi…
            </p>
          )}
          <Button type="button" variant="ghost" className="w-full" onClick={onClose} disabled={busy}>
            Batal
          </Button>
        </div>
      </Modal>
    )
  }

  return (
    <Modal label="Ganti autentikator" onClose={onClose}>
      <form onSubmit={begin} className="space-y-4">
        <div className="flex items-center gap-2.5">
          <span className="flex h-9 w-9 items-center justify-center rounded-xl bg-violet-500/15">
            <Smartphone className="h-4.5 w-4.5 text-violet-400" />
          </span>
          <div>
            <p className="text-base font-semibold leading-tight">Ganti autentikator</p>
            <p className="text-xs text-muted-foreground">Untuk HP baru atau bila HP lama hilang.</p>
          </div>
        </div>
        <ul className="list-disc space-y-1 rounded-lg bg-background/60 py-3 pl-8 pr-3 text-xs text-muted-foreground">
          <li>Kode pemulihan lama hangus dan diganti 8 kode baru.</li>
          <li>Semua perangkat lain dikeluarkan; perangkat ini tetap masuk.</li>
        </ul>
        <StepUpFields password={password} code={code} onPassword={setPassword} onCode={setCode} disabled={busy} />
        {error && <FormError>{error}</FormError>}
        <div className="flex justify-end gap-2">
          <Button type="button" variant="ghost" onClick={onClose} disabled={busy}>
            Batal
          </Button>
          <Button type="submit" disabled={busy || !password || !code.trim()}>
            {busy && <Loader2 className="h-4 w-4 animate-spin" />}
            Lanjut
          </Button>
        </div>
      </form>
    </Modal>
  )
}
