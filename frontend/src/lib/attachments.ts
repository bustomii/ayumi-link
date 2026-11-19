import type { AttachmentKind } from '@/api/types'
import type { UploadLimits } from '@/api/uploads'

/** Cadangan bila server belum menjawab; sama dengan daftar yang diizinkan backend. */
export const DEFAULT_LIMITS: UploadLimits = {
  maxBytes: 20 << 20,
  maxFiles: 5,
  types: [
    { kind: 'image', label: 'Gambar / foto', exts: ['.png', '.jpg', '.jpeg', '.gif', '.webp'] },
    { kind: 'pdf', label: 'PDF', exts: ['.pdf'] },
    { kind: 'excel', label: 'Excel', exts: ['.xlsx', '.csv'] },
    { kind: 'word', label: 'Dokumen Word', exts: ['.docx'] },
  ],
}

const LEGACY: Record<string, string> = {
  '.xls': 'format Excel lama (.xls) tidak didukung; simpan ulang sebagai .xlsx',
  '.doc': 'format Word lama (.doc) tidak didukung; simpan ulang sebagai .docx',
}

export const extOf = (name: string) => {
  const i = name.lastIndexOf('.')
  return i >= 0 ? name.slice(i).toLowerCase() : ''
}

export function kindOfFile(name: string, limits: UploadLimits): AttachmentKind | null {
  const ext = extOf(name)
  return limits.types.find((t) => t.exts.includes(ext))?.kind ?? null
}

/** Nilai atribut `accept` untuk <input type=file>. */
export const acceptAttr = (limits: UploadLimits) => limits.types.flatMap((t) => t.exts).join(',')

export function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(bytes < 10 * 1024 ? 1 : 0)} KB`
  const mb = bytes / (1024 * 1024)
  return `${Number.isInteger(mb) ? mb : mb.toFixed(1)} MB`
}

/** Ringkasan jenis yang diizinkan untuk pesan error/petunjuk. */
export function allowedSummary(limits: UploadLimits): string {
  return limits.types.map((t) => `${t.label} (${t.exts.join(', ')})`).join('; ')
}

export type Screened = { accepted: File[]; problems: string[] }

/**
 * Pemeriksaan awal di browser (cepat, ramah). Server tetap memeriksa isi file yang sebenarnya,
 * jadi ini bukan batas keamanan.
 */
export function screenFiles(files: File[], existing: number, limits: UploadLimits): Screened {
  const accepted: File[] = []
  const problems: string[] = []
  for (const f of files) {
    const ext = extOf(f.name)
    if (!kindOfFile(f.name, limits)) {
      problems.push(`${f.name}: ${LEGACY[ext] ?? 'jenis file tidak diizinkan'}`)
    } else if (f.size === 0) {
      problems.push(`${f.name}: file kosong`)
    } else if (f.size > limits.maxBytes) {
      problems.push(`${f.name}: terlalu besar (${formatSize(f.size)}, maksimal ${formatSize(limits.maxBytes)})`)
    } else if (existing + accepted.length >= limits.maxFiles) {
      problems.push(`${f.name}: maksimal ${limits.maxFiles} lampiran per pesan`)
    } else {
      accepted.push(f)
    }
  }
  return { accepted, problems }
}
