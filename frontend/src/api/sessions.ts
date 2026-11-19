import { api } from './http'
import type { ChangeDiff, DirListing, FileChange, Session, SessionDetail, TermResult } from './types'

export type { DirListing, Message, Session, SessionDetail } from './types'

export async function fetchSessions(): Promise<Session[]> {
  const data = await api<{ sessions: Session[] }>('/api/sessions')
  return data.sessions ?? []
}

export function fetchSessionDetail(
  id: string,
  opts?: { limit?: number; before?: string },
): Promise<SessionDetail> {
  const params = new URLSearchParams({ limit: String(opts?.limit ?? 20) })
  if (opts?.before) params.set('before', opts.before)
  return api<SessionDetail>(`/api/sessions/${encodeURIComponent(id)}?${params}`)
}

export async function moveSession(id: string, group: string): Promise<void> {
  await api(`/api/sessions/${encodeURIComponent(id)}/group`, { method: 'PUT', json: { group } })
}

export async function renameSession(id: string, title: string): Promise<void> {
  await api(`/api/sessions/${encodeURIComponent(id)}/title`, { method: 'PUT', json: { title } })
}

export async function renameGroup(from: string, to: string): Promise<void> {
  await api('/api/groups/rename', { method: 'PUT', json: { from, to } })
}

/** 'deleted' = transcript dihapus permanen; 'hidden' = hanya disembunyikan dari Ayumi Link. */
export async function deleteSession(id: string): Promise<'deleted' | 'hidden'> {
  const data = await api<{ mode: 'deleted' | 'hidden' }>(`/api/sessions/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  })
  return data.mode
}

export function fetchDir(path?: string): Promise<DirListing> {
  const qs = path ? `?path=${encodeURIComponent(path)}` : ''
  return api<DirListing>(`/api/fs${qs}`)
}

export async function fetchSessionChanges(id: string): Promise<FileChange[]> {
  const data = await api<{ changes: FileChange[] }>(`/api/sessions/${encodeURIComponent(id)}/changes`)
  return data.changes ?? []
}

export function fetchSessionChangeDiff(id: string, path: string): Promise<ChangeDiff> {
  const qs = `?path=${encodeURIComponent(path)}`
  return api<ChangeDiff>(`/api/sessions/${encodeURIComponent(id)}/changes/diff${qs}`)
}

/** Satu shell global: masuk ke folder chat (atau root AYUMI_ROOTS). */
export function enterTerminal(sessionId?: string): Promise<{ cwd: string }> {
  const qs = sessionId ? `?sessionId=${encodeURIComponent(sessionId)}` : ''
  return api(`/api/terminal${qs}`)
}

export function runTerminal(command: string): Promise<TermResult> {
  return api('/api/terminal', { method: 'POST', json: { command } })
}

/** Menyalin session ke folder kerja baru (session baru, yang lama tidak diubah). */
export function forkSession(id: string, cwd: string): Promise<{ id: string; cwd: string }> {
  return api(`/api/sessions/${encodeURIComponent(id)}/fork`, { method: 'POST', json: { cwd } })
}
