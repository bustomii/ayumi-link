import { api } from './http'

export type ToolTab = {
  id: string
  sessionId?: string
  title: string
  cwd?: string
  url?: string
  active?: boolean
  alive?: boolean
  createdAt?: string
  updatedAt?: string
}

export type ToolsTabs = {
  terminal: ToolTab[]
  browser: ToolTab[]
}

export const fetchToolsTabs = (sessionId: string) =>
  api<ToolsTabs>(`/api/sessions/${encodeURIComponent(sessionId)}/tools/tabs`)

export const createTerminalTab = (sessionId: string, title?: string) =>
  api<ToolTab>(`/api/sessions/${encodeURIComponent(sessionId)}/terminal/tabs`, {
    method: 'POST',
    json: { title: title || 'Terminal' },
  })

export const deleteTerminalTab = (sessionId: string, tabId: string) =>
  api<{ ok: boolean }>(
    `/api/sessions/${encodeURIComponent(sessionId)}/terminal/tabs/${encodeURIComponent(tabId)}`,
    { method: 'DELETE' },
  )

export const createBrowserTab = (sessionId: string, body?: { title?: string; url?: string }) =>
  api<ToolTab>(`/api/sessions/${encodeURIComponent(sessionId)}/browser/tabs`, {
    method: 'POST',
    json: body ?? {},
  })

export const deleteBrowserTab = (sessionId: string, tabId: string) =>
  api<{ ok: boolean }>(
    `/api/sessions/${encodeURIComponent(sessionId)}/browser/tabs/${encodeURIComponent(tabId)}`,
    { method: 'DELETE' },
  )

export const activateBrowserTab = (sessionId: string, tabId: string) =>
  api<ToolTab>(
    `/api/sessions/${encodeURIComponent(sessionId)}/browser/tabs/${encodeURIComponent(tabId)}/activate`,
    { method: 'POST', json: {} },
  )

export function terminalWsUrl(sessionId: string, tabId: string) {
  const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${proto}//${window.location.host}/api/terminal/ws?sessionId=${encodeURIComponent(sessionId)}&tabId=${encodeURIComponent(tabId)}`
}
