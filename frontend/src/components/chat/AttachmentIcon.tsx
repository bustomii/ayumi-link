import { File, FileImage, FileSpreadsheet, FileText, type LucideIcon } from 'lucide-react'
import { cn } from '@/lib/utils'

const STYLE: Record<string, { icon: LucideIcon; tone: string; bg: string }> = {
  image: { icon: FileImage, tone: 'text-sky-400', bg: 'bg-sky-500/15' },
  pdf: { icon: FileText, tone: 'text-red-400', bg: 'bg-red-500/15' },
  excel: { icon: FileSpreadsheet, tone: 'text-emerald-400', bg: 'bg-emerald-500/15' },
  word: { icon: FileText, tone: 'text-blue-400', bg: 'bg-blue-500/15' },
}

/** Ikon berwarna sesuai jenis lampiran. */
export function AttachmentIcon({ kind, className }: { kind: string; className?: string }) {
  const s = STYLE[kind] ?? { icon: File, tone: 'text-muted-foreground', bg: 'bg-muted' }
  const Icon = s.icon
  return (
    <span className={cn('flex h-9 w-9 shrink-0 items-center justify-center rounded-lg', s.bg, className)}>
      <Icon className={cn('h-4.5 w-4.5', s.tone)} />
    </span>
  )
}
