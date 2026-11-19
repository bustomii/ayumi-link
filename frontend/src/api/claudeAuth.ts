import { api } from './http'

export type ClaudeAuthStatus = {
  loggedIn: boolean
  authMethod?: string
  stored: boolean
  storedMethod?: string
  tokenHint?: string
  loginExpiresAt?: string
  source?: string
  message?: string
}

export const fetchClaudeAuth = () => api<ClaudeAuthStatus>('/api/claude/auth')

export const saveClaudeAuth = (body: { method: 'oauth_token' | 'api_key'; token: string }) =>
  api<ClaudeAuthStatus>('/api/claude/auth', { method: 'PUT', json: body })

export const clearClaudeAuth = () => api<ClaudeAuthStatus>('/api/claude/auth', { method: 'DELETE' })
