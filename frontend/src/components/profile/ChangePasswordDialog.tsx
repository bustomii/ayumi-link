import { CheckCircle2, KeyRound, Loader2 } from 'lucide-react'
import { useState, type FormEvent } from 'react'
import { changePassword } from '@/api/profile'
import { Field, FormError, PasswordInput } from '@/components/auth/Field'
import { formErrorText } from '@/components/profile/form-errors'
import { StepUpFields } from '@/components/profile/StepUpFields'
import { Button } from '@/components/ui/button'
import { Modal } from '@/components/ui/modal'

const MIN_PASSWORD = 12

type Props = { onClose: () => void }

export function ChangePasswordDialog({ onClose }: Props) {
  const [current, setCurrent] = useState('')
  const [code, setCode] = useState('')
  const [next, setNext] = useState('')
  const [confirm, setConfirm] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [done, setDone] = useState(false)

  const mismatch = confirm.length > 0 && confirm !== next
  const valid = current && code.trim() && next.length >= MIN_PASSWORD && next === confirm

  async function submit(e: FormEvent) {
    e.preventDefault()
    if (busy || !valid) return
    setBusy(true)
    setError(null)
    try {
      await changePassword({ currentPassword: current, newPassword: next, code: code.trim() })
      setDone(true)
    } catch (err) {
      setCode('')
      setError(formErrorText(err))
    } finally {
      setBusy(false)
    }
  }

  if (done) {
    return (
      <Modal label="Password diganti" onClose={onClose} className="max-w-sm text-center">
        <span className="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-emerald-500/15">
          <CheckCircle2 className="h-6 w-6 text-emerald-400" />
        </span>
        <h2 className="mt-3 text-base font-semibold">Password berhasil diganti</h2>
        <p className="mt-1.5 text-sm text-muted-foreground">
          Semua perangkat lain dikeluarkan. Perangkat ini tetap masuk.
        </p>
        <Button className="mt-5 w-full" onClick={onClose} autoFocus>
          Selesai
        </Button>
      </Modal>
    )
  }

  return (
    <Modal label="Ubah password" onClose={onClose}>
      <form onSubmit={submit} className="space-y-4">
        <div className="flex items-center gap-2.5">
          <span className="flex h-9 w-9 items-center justify-center rounded-xl bg-sky-500/15">
            <KeyRound className="h-4.5 w-4.5 text-sky-400" />
          </span>
          <div>
            <p className="text-base font-semibold leading-tight">Ubah password</p>
            <p className="text-xs text-muted-foreground">Perangkat lain akan dikeluarkan.</p>
          </div>
        </div>
        <StepUpFields password={current} code={code} onPassword={setCurrent} onCode={setCode} disabled={busy} />
        <Field label="Password baru" hint={`Minimal ${MIN_PASSWORD} karakter dan berbeda dari yang lama.`}>
          <PasswordInput value={next} onChange={(e) => setNext(e.target.value)} autoComplete="new-password" required />
        </Field>
        <Field label="Ulangi password baru">
          <PasswordInput value={confirm} onChange={(e) => setConfirm(e.target.value)} autoComplete="new-password" required />
        </Field>
        {mismatch && <FormError>Password baru tidak sama.</FormError>}
        {error && <FormError>{error}</FormError>}
        <div className="flex justify-end gap-2">
          <Button type="button" variant="ghost" onClick={onClose} disabled={busy}>
            Batal
          </Button>
          <Button type="submit" disabled={busy || !valid}>
            {busy && <Loader2 className="h-4 w-4 animate-spin" />}
            Simpan
          </Button>
        </div>
      </form>
    </Modal>
  )
}
