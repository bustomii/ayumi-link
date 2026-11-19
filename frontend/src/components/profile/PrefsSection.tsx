import { SlidersHorizontal } from 'lucide-react'
import type { ReactNode } from 'react'
import { Section } from '@/components/profile/Section'
import { EFFORTS, MODELS, MORE_MODELS, PERMISSION_MODES, type Option, type Prefs } from '@/lib/chat-options'

type Props = { prefs: Prefs; onPrefs: (p: Prefs) => void }

const selectClass =
  'w-full rounded-lg border border-border bg-background px-3 py-2 text-sm outline-none focus:border-primary sm:max-w-xs'

function Row({ label, hint, children }: { label: string; hint: string; children: ReactNode }) {
  return (
    <div className="flex flex-col gap-1.5 py-3 sm:flex-row sm:items-center sm:justify-between sm:gap-4">
      <div>
        <p className="text-sm font-medium">{label}</p>
        <p className="text-xs text-muted-foreground">{hint}</p>
      </div>
      {children}
    </div>
  )
}

const options = (list: Option[]) =>
  list.map((o) => (
    <option key={o.value} value={o.value}>
      {o.label} — {o.hint}
    </option>
  ))

/** Pengaturan bawaan untuk pesan baru; tersimpan di server dan berlaku di semua perangkat. */
export function PrefsSection({ prefs, onPrefs }: Props) {
  return (
    <Section
      icon={SlidersHorizontal}
      tone="text-fuchsia-400"
      title="Pengaturan chat bawaan"
      description="Dipakai untuk pesan baru. Bisa diubah sementara lewat menu di kotak pesan."
    >
      <div className="divide-y divide-border/50">
        <Row label="Model" hint="Model Claude yang menjawab.">
          <select className={selectClass} value={prefs.model} onChange={(e) => onPrefs({ ...prefs, model: e.target.value })}>
            <optgroup label="Terbaru">{options(MODELS)}</optgroup>
            <optgroup label="Model lainnya">{options(MORE_MODELS)}</optgroup>
          </select>
        </Row>
        <Row label="Effort" hint="Seberapa dalam Claude berpikir.">
          <select className={selectClass} value={prefs.effort} onChange={(e) => onPrefs({ ...prefs, effort: e.target.value })}>
            {options(EFFORTS)}
          </select>
        </Row>
        <Row label="Izin tool" hint="Apakah Claude boleh menjalankan tool otomatis.">
          <select
            className={selectClass}
            value={prefs.permissionMode}
            onChange={(e) => onPrefs({ ...prefs, permissionMode: e.target.value })}
          >
            {options(PERMISSION_MODES)}
          </select>
        </Row>
      </div>
    </Section>
  )
}
