import type { Session } from '@/api/sessions'

export type SessionGroup = {
  key: string
  label: string
  path: string
  sessions: Session[]
  liveCount: number
  updatedAt: number
  order: number
}

/**
 * Group using Claude Desktop custom groups (`session.group` from API).
 * Fallback label = folder basename already resolved on backend.
 */
export function groupSessionsByFolder(sessions: Session[]): SessionGroup[] {
  const map = new Map<string, Session[]>()
  for (const s of sessions) {
    const key = (s.group || s.project || 'Lainnya').trim() || 'Lainnya'
    const list = map.get(key) ?? []
    list.push(s)
    map.set(key, list)
  }

  const groups: SessionGroup[] = []
  for (const [key, list] of map) {
    const sorted = [...list].sort(
      (a, b) => new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime(),
    )
    const updatedAt = Math.max(...sorted.map((s) => new Date(s.updatedAt).getTime() || 0))
    const order = Math.min(...sorted.map((s) => s.groupOrder ?? 9999))
    const sample = sorted[0]
    groups.push({
      key,
      label: key,
      path: sample?.cwd?.replace(/^\/Users\/[^/]+/, '~') || '',
      sessions: sorted,
      liveCount: sorted.filter((s) => s.live).length,
      updatedAt,
      order,
    })
  }

  groups.sort((a, b) => {
    if (a.order !== b.order) return a.order - b.order
    return b.updatedAt - a.updatedAt
  })
  return groups
}
