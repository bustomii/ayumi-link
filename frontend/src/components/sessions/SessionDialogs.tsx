import { useState } from 'react'
import type { Session } from '@/api/sessions'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Modal } from '@/components/ui/modal'
import { cn } from '@/lib/utils'

export function MoveSessionDialog({
  session,
  groups,
  onCancel,
  onMove,
}: {
  session: Session
  groups: string[]
  onCancel: () => void
  onMove: (group: string) => void
}) {
  const [name, setName] = useState('')
  const current = session.group ?? ''
  return (
    <Modal label="Pindah ke folder" onClose={onCancel}>
      <p className="text-base font-semibold">Pindah ke folder</p>
      <p className="mt-1 truncate text-xs text-muted-foreground">{session.title}</p>
      <div className="mt-3 max-h-56 space-y-1 overflow-y-auto">
        {groups.map((g) => (
          <button
            key={g}
            type="button"
            disabled={g === current}
            onClick={() => onMove(g)}
            className={cn(
              'flex w-full items-center justify-between rounded-lg px-3 py-2 text-left text-sm transition-colors hover:bg-white/5 disabled:opacity-50',
              g === current && 'bg-white/10',
            )}
          >
            <span className="truncate">{g}</span>
            {g === current && <span className="text-[11px] text-muted-foreground">sekarang</span>}
          </button>
        ))}
      </div>
      <form
        className="mt-3 flex gap-2"
        onSubmit={(e) => {
          e.preventDefault()
          if (name.trim()) onMove(name.trim())
        }}
      >
        <Input
          value={name}
          maxLength={60}
          onChange={(e) => setName(e.target.value)}
          placeholder="Atau buat folder baru…"
          className="min-w-0 flex-1"
        />
        <Button type="submit" disabled={!name.trim()}>
          Pindah
        </Button>
      </form>
      <p className="mt-3 text-[11px] text-muted-foreground">
        Perubahan folder hanya berlaku di Ayumi Link, tidak mengubah pengelompokan di Claude Desktop.
      </p>
      <div className="mt-3 flex justify-end">
        <Button variant="ghost" onClick={onCancel}>
          Batal
        </Button>
      </div>
    </Modal>
  )
}

export function DeleteSessionDialog({
  session,
  busy,
  onCancel,
  onConfirm,
}: {
  session: Session
  busy: boolean
  onCancel: () => void
  onConfirm: () => void
}) {
  const local = Boolean(session.local)
  return (
    <Modal label={local ? 'Hapus session' : 'Sembunyikan session'} onClose={onCancel}>
      <p className="text-base font-semibold">{local ? 'Hapus session?' : 'Sembunyikan session?'}</p>
      <p className="mt-1 truncate text-sm">{session.title}</p>
      <p className="mt-3 text-sm text-muted-foreground">
        {local
          ? 'Session ini dibuat lewat Ayumi Link. Seluruh isi percakapannya akan dihapus permanen dari Mac dan tidak bisa dikembalikan.'
          : 'Session ini milik Claude Desktop. Di Ayumi Link ia hanya disembunyikan; percakapan dan data Desktop tidak diubah. Untuk menghapus permanen, hapus di Claude Desktop.'}
      </p>
      <div className="mt-4 flex justify-end gap-2">
        <Button variant="ghost" onClick={onCancel} disabled={busy}>
          Batal
        </Button>
        <Button onClick={onConfirm} disabled={busy} className={local ? 'bg-red-600 text-white hover:bg-red-500' : ''}>
          {busy ? 'Memproses…' : local ? 'Hapus permanen' : 'Sembunyikan'}
        </Button>
      </div>
    </Modal>
  )
}

type RenameProps = {
  heading: string
  description: string
  initial: string
  maxLength: number
  /** Boleh kosong (mengembalikan nama asli)? */
  allowEmpty?: boolean
  busy: boolean
  onCancel: () => void
  onSubmit: (value: string) => void
}

/** Dialog satu kolom untuk mengubah nama session atau folder. */
export function RenameDialog({
  heading,
  description,
  initial,
  maxLength,
  allowEmpty = false,
  busy,
  onCancel,
  onSubmit,
}: RenameProps) {
  const [value, setValue] = useState(initial)
  const trimmed = value.trim()
  const canSave = !busy && (allowEmpty || trimmed.length > 0) && trimmed !== initial.trim()
  return (
    <Modal label={heading} onClose={onCancel}>
      <form
        onSubmit={(e) => {
          e.preventDefault()
          if (canSave) onSubmit(trimmed)
        }}
      >
        <p className="text-base font-semibold">{heading}</p>
        <p className="mt-1 text-xs text-muted-foreground">{description}</p>
        <Input
          autoFocus
          value={value}
          maxLength={maxLength}
          onChange={(e) => setValue(e.target.value)}
          onFocus={(e) => e.currentTarget.select()}
          className="mt-3"
          aria-label={heading}
        />
        <div className="mt-4 flex justify-end gap-2">
          <Button type="button" variant="ghost" onClick={onCancel} disabled={busy}>
            Batal
          </Button>
          <Button type="submit" disabled={!canSave}>
            {busy ? 'Menyimpan…' : 'Simpan'}
          </Button>
        </div>
      </form>
    </Modal>
  )
}
