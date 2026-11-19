/** Ringkas string User-Agent jadi "Browser · Sistem" yang mudah dibaca. */
export function describeUA(ua: string): string {
  if (!ua) return 'Perangkat tidak dikenal'
  const browser =
    /Edg\//.test(ua) ? 'Edge'
    : /OPR\/|Opera/.test(ua) ? 'Opera'
    : /Firefox\//.test(ua) ? 'Firefox'
    : /Chrome\//.test(ua) || /CriOS\//.test(ua) ? 'Chrome'
    : /Safari\//.test(ua) ? 'Safari'
    : /curl\//i.test(ua) ? 'curl'
    : 'Browser'
  const os =
    /iPhone|iPad|iPod/.test(ua) ? 'iOS'
    : /Android/.test(ua) ? 'Android'
    : /Mac OS X|Macintosh/.test(ua) ? 'macOS'
    : /Windows/.test(ua) ? 'Windows'
    : /Linux/.test(ua) ? 'Linux'
    : ''
  return os ? `${browser} · ${os}` : browser
}

const rtf = new Intl.RelativeTimeFormat('id-ID', { numeric: 'auto' })

/** "baru saja", "5 menit lalu", "kemarin", … */
export function timeAgo(iso: string, now = Date.now()): string {
  const t = new Date(iso).getTime()
  if (Number.isNaN(t)) return ''
  const sec = Math.round((t - now) / 1000)
  if (Math.abs(sec) < 45) return 'baru saja'
  const units: [Intl.RelativeTimeFormatUnit, number][] = [
    ['day', 86400],
    ['hour', 3600],
    ['minute', 60],
  ]
  for (const [unit, size] of units) {
    if (Math.abs(sec) >= size) return rtf.format(Math.round(sec / size), unit)
  }
  return rtf.format(sec, 'second')
}

export function formatDate(iso: string): string {
  const d = new Date(iso)
  return Number.isNaN(d.getTime())
    ? ''
    : d.toLocaleDateString('id-ID', { day: 'numeric', month: 'long', year: 'numeric' })
}
