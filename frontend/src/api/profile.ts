import type { Prefs } from '@/lib/chat-options'
import type { PwaBranding } from '@/lib/pwa-branding'
import type { SetupInfo } from './auth'
import { api } from './http'

export type LoginSession = {
  id: string
  createdAt: string
  lastSeen: string
  ua: string
  ip: string
  current: boolean
}

export type ConfigInfo = {
  version: string
  address: string
  roots: string[]
  claudeBin: string
  claudeVersion: string
  dataDir: string
  cookieSecure: boolean
  trustProxy: boolean
  permissionMode: string
}

export type ProfileData = {
  username: string
  createdAt: string
  recoveryRemaining: number
  /** Sejak kapan autentikator saat ini dipakai. */
  twoFactorSince: string
  sessions: LoginSession[]
  prefs: Prefs
  config: ConfigInfo
}

export const fetchProfile = () => api<ProfileData>('/api/profile')

export const fetchPrefs = () => api<Prefs>('/api/profile/prefs')

export const savePrefs = (prefs: Prefs) => api<Prefs>('/api/profile/prefs', { method: 'PUT', json: prefs })

export const fetchBranding = () => api<PwaBranding>('/api/profile/branding')

export const saveBranding = (body: { icon: string; bg: string; fg: string }) =>
  api<PwaBranding>('/api/profile/branding', { method: 'PUT', json: body })

export const resetBranding = () => api<PwaBranding>('/api/profile/branding', { method: 'DELETE' })

export const changePassword = (body: { currentPassword: string; newPassword: string; code: string }) =>
  api<{ ok: true }>('/api/profile/password', { method: 'POST', json: body })

export const regenerateRecovery = (body: { password: string; code: string }) =>
  api<{ ok: true; recoveryCodes: string[] }>('/api/profile/recovery', { method: 'POST', json: body })

export const revokeSession = (id: string) =>
  api<{ ok: true }>(`/api/profile/sessions/${encodeURIComponent(id)}`, { method: 'DELETE' })

export const revokeOtherSessions = () =>
  api<{ ok: true; revoked: number }>('/api/profile/sessions/revoke-others', { method: 'POST', json: {} })

/** Langkah 1 mengganti autentikator: verifikasi ulang password + kode saat ini. */
export const beginTwoFactorChange = (body: { password: string; code: string }) =>
  api<SetupInfo>('/api/profile/2fa/begin', { method: 'POST', json: body })

/** Langkah 2: kode dari perangkat baru. Mengembalikan kode pemulihan baru (ditampilkan sekali). */
export const confirmTwoFactorChange = (body: { code: string }) =>
  api<{ ok: true; recoveryCodes: string[] }>('/api/profile/2fa/confirm', { method: 'POST', json: body })
