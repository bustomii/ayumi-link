import { api } from './http'
import type { AuthStatus } from './types'

export type SetupInfo = { secret: string; otpauthUrl: string; qr: string }

export const fetchAuthStatus = () => api<AuthStatus>('/api/auth/status')

export const beginSetup = (body: { setupToken: string; username: string; password: string }) =>
  api<SetupInfo>('/api/auth/setup', { method: 'POST', json: body })

export const confirmSetup = (body: { setupToken: string; code: string }) =>
  api<{ ok: true; recoveryCodes: string[] }>('/api/auth/setup/confirm', { method: 'POST', json: body })

export const login = (body: { username: string; password: string }) =>
  api<{ mfaRequired: true; mfaToken: string }>('/api/auth/login', { method: 'POST', json: body })

export const verifyLogin = (body: { mfaToken: string; code: string }) =>
  api<{ ok: true }>('/api/auth/login/verify', { method: 'POST', json: body })

export const logout = () => api<{ ok: true }>('/api/auth/logout', { method: 'POST', json: {} })
