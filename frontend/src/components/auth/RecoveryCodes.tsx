import { Check, Copy, Download } from 'lucide-react'
import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { copyText, downloadText } from '@/lib/clipboard'

type Props = {
  codes: string[]
  doneLabel: string
  onDone: () => void
}

/** Tampilkan kode pemulihan sekali; lanjut baru aktif setelah pengguna menyatakan sudah menyimpannya. */
export function RecoveryCodes({ codes, doneLabel, onDone }: Props) {
  const [saved, setSaved] = useState(false)
  const [copied, setCopied] = useState(false)
  const text = codes.join('\n')

  return (
    <div className="space-y-4">
      <ul className="grid grid-cols-2 gap-2 rounded-lg bg-background/60 p-3 font-mono text-sm">
        {codes.map((c) => (
          <li key={c} className="text-center tracking-wider">
            {c}
          </li>
        ))}
      </ul>
      <div className="flex gap-2">
        <Button
          type="button"
          variant="outline"
          className="flex-1"
          onClick={async () => {
            if (await copyText(text)) {
              setCopied(true)
              setTimeout(() => setCopied(false), 1500)
            }
          }}
        >
          {copied ? <Check className="h-4 w-4 text-emerald-400" /> : <Copy className="h-4 w-4" />}
          Salin
        </Button>
        <Button
          type="button"
          variant="outline"
          className="flex-1"
          onClick={() => downloadText('ayumi-link-recovery-codes.txt', `Ayumi Link - kode pemulihan\n\n${text}\n`)}
        >
          <Download className="h-4 w-4" />
          Unduh
        </Button>
      </div>
      <label className="flex items-start gap-2 text-sm">
        <input
          type="checkbox"
          checked={saved}
          onChange={(e) => setSaved(e.target.checked)}
          className="mt-0.5 h-4 w-4 accent-[#3d8bfd]"
        />
        <span>Saya sudah menyimpan kode pemulihan di tempat yang aman.</span>
      </label>
      <Button type="button" className="w-full" disabled={!saved} onClick={onDone}>
        {doneLabel}
      </Button>
    </div>
  )
}
