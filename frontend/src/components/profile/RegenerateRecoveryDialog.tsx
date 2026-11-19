import { Loader2, LifeBuoy } from 'lucide-react'
import { useState, type FormEvent } from 'react'
import { regenerateRecovery } from '@/api/profile'
import { RecoveryCodes } from '@/components/auth/RecoveryCodes'
import { FormError } from '@/components/auth/Field'
import { formErrorText } from '@/components/profile/form-errors'
import { StepUpFields } from '@/components/profile/StepUpFields'
import { Button } from '@/components/ui/button'
import { Modal } from '@/components/ui/modal'

type Props = { onClose: () => void; onDone: () => void }

export function RegenerateRecoveryDialog({ onClose, onDone }: Props) {
  const [password, setPassword] = useState('')
  const [code, setCode] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [codes, setCodes] = useState<string[] | null>(null)

  async function submit(e: FormEvent) {
    e.preventDefault()
    if (busy) return
    setBusy(true)
    setError(null)
    try {
      const res = await regenerateRecovery({ password, code: code.trim() })
      setCodes(res.recoveryCodes)
      onDone()
    } catch (err) {
      setCode('')
      setError(formErrorText(err))
    } finally {
      setBusy(false)
    }
  }

  if (codes) {
    // Tidak bisa ditutup lewat latar/Escape: kode hanya tampil sekali.
    return (
      <Modal label="Kode pemulihan baru" onClose={() => {}}>
        <p className="text-base font-semibold">Kode pemulihan baru</p>
        <p className="mb-4 mt-1 text-xs text-muted-foreground">
          Kode lama sudah hangus. Simpan kode ini; tidak akan ditampilkan lagi.
        </p>
        <RecoveryCodes codes={codes} doneLabel="Selesai" onDone={onClose} />
      </Modal>
    )
  }

  return (
    <Modal label="Buat ulang kode pemulihan" onClose={onClose}>
      <form onSubmit={submit} className="space-y-4">
        <div className="flex items-center gap-2.5">
          <span className="flex h-9 w-9 items-center justify-center rounded-xl bg-amber-500/15">
            <LifeBuoy className="h-4.5 w-4.5 text-amber-400" />
          </span>
          <div>
            <p className="text-base font-semibold leading-tight">Buat ulang kode pemulihan</p>
            <p className="text-xs text-muted-foreground">8 kode baru; semua kode lama hangus.</p>
          </div>
        </div>
        <StepUpFields password={password} code={code} onPassword={setPassword} onCode={setCode} disabled={busy} />
        {error && <FormError>{error}</FormError>}
        <div className="flex justify-end gap-2">
          <Button type="button" variant="ghost" onClick={onClose} disabled={busy}>
            Batal
          </Button>
          <Button type="submit" disabled={busy || !password || !code.trim()}>
            {busy && <Loader2 className="h-4 w-4 animate-spin" />}
            Buat kode baru
          </Button>
        </div>
      </form>
    </Modal>
  )
}
