import { CheckCircle2, KeyRound, Loader2, LogIn, Trash2, XCircle } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import { clearClaudeAuth, fetchClaudeAuth, saveClaudeAuth, type ClaudeAuthStatus } from '@/api/claudeAuth'
import { InfoRow, Section } from '@/components/profile/Section'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { cn } from '@/lib/utils'

type Method = 'oauth_token' | 'api_key'

/** Login Claude CLI (Docker/container) dari halaman profil. */
export function ClaudeAuthSection({ onError }: { onError: (e: unknown, action: string) => void }) {
  const [status, setStatus] = useState<ClaudeAuthStatus | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [method, setMethod] = useState<Method>('oauth_token')
  const [token, setToken] = useState('')
  const [showForm, setShowForm] = useState(false)

  const reload = useCallback(async () => {
    setLoading(true)
    try {
      const s = await fetchClaudeAuth()
      setStatus(s)
      if (!s.loggedIn) setShowForm(true)
    } catch (e) {
      onError(e, 'Memuat status Claude CLI')
    } finally {
      setLoading(false)
    }
  }, [onError])

  useEffect(() => {
    void reload()
  }, [reload])

  const onSave = async () => {
    setSaving(true)
    try {
      const s = await saveClaudeAuth({ method, token: token.trim() })
      setStatus(s)
      setToken('')
      setShowForm(!s.loggedIn)
    } catch (e) {
      onError(e, 'Menyimpan login Claude')
    } finally {
      setSaving(false)
    }
  }

  const onClear = async () => {
    setSaving(true)
    try {
      const s = await clearClaudeAuth()
      setStatus(s)
      setShowForm(true)
    } catch (e) {
      onError(e, 'Menghapus login Claude')
    } finally {
      setSaving(false)
    }
  }

  const loggedIn = Boolean(status?.loggedIn)

  return (
    <Section
      icon={KeyRound}
      tone="text-violet-400"
      title="Claude CLI"
      description="Login akun Claude untuk chat di container/server ini."
      aside={
        loading ? (
          <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
        ) : loggedIn ? (
          <span className="inline-flex items-center gap-1 rounded-full bg-emerald-500/15 px-2 py-0.5 text-[11px] font-medium text-emerald-400">
            <CheckCircle2 className="h-3.5 w-3.5" /> Login
          </span>
        ) : (
          <span className="inline-flex items-center gap-1 rounded-full bg-red-500/15 px-2 py-0.5 text-[11px] font-medium text-red-400">
            <XCircle className="h-3.5 w-3.5" /> Belum login
          </span>
        )
      }
    >
      <dl className="divide-y divide-border/50">
        <InfoRow label="Status">{loading ? '…' : loggedIn ? 'Sudah login' : 'Belum login'}</InfoRow>
        {status?.authMethod && status.authMethod !== 'none' && (
          <InfoRow label="Metode">{status.authMethod}</InfoRow>
        )}
        <InfoRow label="Sumber">
          {status?.source === 'stored'
            ? `Disimpan di Ayumi${status.tokenHint ? ` (${status.tokenHint})` : ''}`
            : status?.source === 'env'
              ? 'Environment (.env)'
              : 'Tidak ada'}
        </InfoRow>
        {status?.message && !loggedIn && (
          <InfoRow label="Catatan">
            <span className="text-amber-300/90">{status.message}</span>
          </InfoRow>
        )}
      </dl>

      <div className="mt-4 flex flex-wrap gap-2">
        <Button variant="outline" size="sm" onClick={() => void reload()} disabled={loading || saving}>
          Refresh status
        </Button>
        {!showForm && (
          <Button variant="outline" size="sm" onClick={() => setShowForm(true)} disabled={saving}>
            <LogIn className="h-3.5 w-3.5" />
            Ganti token
          </Button>
        )}
        {status?.stored && (
          <Button variant="outline" size="sm" onClick={() => void onClear()} disabled={saving}>
            <Trash2 className="h-3.5 w-3.5" />
            Hapus token tersimpan
          </Button>
        )}
      </div>

      {showForm && (
        <div className="mt-4 space-y-3 rounded-xl border border-border/60 bg-background/40 p-3">
          <p className="text-xs text-muted-foreground">
            Di Mac jalankan <code className="rounded bg-muted px-1 py-0.5 font-mono text-[11px]">claude setup-token</code>,
            salin token, lalu tempel di sini. Atau pakai API key Anthropic.
          </p>
          <div className="flex gap-2">
            {(
              [
                ['oauth_token', 'Setup token'],
                ['api_key', 'API key'],
              ] as const
            ).map(([id, label]) => (
              <button
                key={id}
                type="button"
                onClick={() => setMethod(id)}
                className={cn(
                  'rounded-lg border px-3 py-1.5 text-xs transition-colors',
                  method === id ? 'border-primary bg-primary/15 text-foreground' : 'border-border text-muted-foreground hover:bg-white/5',
                )}
              >
                {label}
              </button>
            ))}
          </div>
          <Input
            type="password"
            autoComplete="off"
            spellCheck={false}
            placeholder={method === 'api_key' ? 'sk-ant-api…' : 'Token dari claude setup-token'}
            value={token}
            onChange={(e) => setToken(e.target.value)}
            disabled={saving}
          />
          <div className="flex flex-wrap gap-2">
            <Button onClick={() => void onSave()} disabled={saving || token.trim().length < 20}>
              {saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <LogIn className="h-4 w-4" />}
              Simpan & verifikasi
            </Button>
            {loggedIn && (
              <Button variant="ghost" onClick={() => { setShowForm(false); setToken('') }} disabled={saving}>
                Batal
              </Button>
            )}
          </div>
        </div>
      )}
    </Section>
  )
}
