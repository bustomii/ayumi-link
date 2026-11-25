import { api } from './http'

export type GitFile = { path: string; status: string; fromSession: boolean }

export type GitRepo = {
  path: string
  name: string
  branch?: string
  setup: boolean
  files: GitFile[]
  error?: string
}

const base = (id: string) => `/api/sessions/${encodeURIComponent(id)}/git`

export async function fetchRepos(id: string): Promise<GitRepo[]> {
  const data = await api<{ repos: GitRepo[] }>(base(id))
  return data.repos ?? []
}

export function fetchRepoDiff(id: string, repo: string, path: string) {
  return api<{ diff: string }>(`${base(id)}/diff?repo=${encodeURIComponent(repo)}&path=${encodeURIComponent(path)}`)
}

export function addRepo(id: string, path: string) {
  return api<{ ok: boolean }>(`${base(id)}/repos`, { method: 'POST', json: { path } })
}

export function removeRepo(id: string, path: string) {
  return api<{ ok: boolean }>(`${base(id)}/repos?path=${encodeURIComponent(path)}`, { method: 'DELETE' })
}
