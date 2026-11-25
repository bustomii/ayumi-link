import { Check, ImageIcon, Loader2, RotateCcw } from 'lucide-react'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { fetchBranding, resetBranding, saveBranding } from '@/api/profile'
import { Section } from '@/components/profile/Section'
import { Button } from '@/components/ui/button'
import {
  applyPwaBrandingLinks,
  brandingPreviewSvg,
  PWA_BG_PRESETS,
  PWA_ICON_OPTIONS,
  type PwaBranding,
  type PwaIconId,
} from '@/lib/pwa-branding'
import { cn } from '@/lib/utils'

type Props = { onError: (e: unknown, action: string) => void }

function Preview({ icon, bg, fg }: { icon: string; bg: string; fg: string }) {
  const svg = useMemo(() => brandingPreviewSvg(icon, bg, fg, 96), [icon, bg, fg])
  return (
    <img
      src={`data:image/svg+xml;utf8,${encodeURIComponent(svg)}`}
      alt="Pratinjau ikon PWA"
      className="h-20 w-20 rounded-2xl border border-border/60 shadow-sm"
      width={80}
      height={80}
    />
  )
}

/** Pilih ikon + warna latar untuk PWA / home screen. */
export function PwaIconSection({ onError }: Props) {
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [icon, setIcon] = useState<PwaIconId>('ayumi')
  const [bg, setBg] = useState('#0c0d10')
  const [fg, setFg] = useState('#ffffff')
  const [saved, setSaved] = useState<PwaBranding | null>(null)

  const apply = useCallback((b: PwaBranding) => {
    setIcon((b.icon as PwaIconId) || 'ayumi')
    setBg(b.bg || '#0c0d10')
    setFg(b.fg || '#ffffff')
    setSaved(b)
    applyPwaBrandingLinks(b)
  }, [])

  const reload = useCallback(async () => {
    setLoading(true)
    try {
      apply(await fetchBranding())
    } catch (e) {
      onError(e, 'Memuat ikon PWA')
    } finally {
      setLoading(false)
    }
  }, [apply, onError])

  useEffect(() => {
    void reload()
  }, [reload])

  const dirty =
    !saved || saved.icon !== icon || saved.bg.toLowerCase() !== bg.toLowerCase() || saved.fg.toLowerCase() !== fg.toLowerCase()

  const onSave = async () => {
    setSaving(true)
    try {
      apply(await saveBranding({ icon, bg, fg }))
    } catch (e) {
      onError(e, 'Menyimpan ikon PWA')
    } finally {
      setSaving(false)
    }
  }

  const onReset = async () => {
    setSaving(true)
    try {
      apply(await resetBranding())
    } catch (e) {
      onError(e, 'Mengembalikan ikon bawaan')
    } finally {
      setSaving(false)
    }
  }

  const lightBg = /^#(f|e|d)/i.test(bg)
  const autoFg = lightBg ? '#0c0d10' : '#ffffff'

  return (
    <Section
      icon={ImageIcon}
      tone="text-sky-400"
      title="Ikon PWA"
      description="Ikon di home screen (iOS/Android) dan favicon. Latar bisa diganti."
      aside={
        loading ? <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" /> : null
      }
    >
      <div className="flex flex-col gap-4 sm:flex-row sm:items-start">
        <Preview icon={icon} bg={bg} fg={fg || autoFg} />
        <div className="min-w-0 flex-1 space-y-3">
          <p className="text-xs text-muted-foreground">
            Setelah disimpan, tab browser langsung berubah. Di HP yang sudah Add to Home Screen,
            biasanya perlu pasang ulang agar ikon baru muncul.
          </p>
          <div>
            <p className="mb-1.5 text-[11px] font-medium text-muted-foreground">Ikon</p>
            <div className="grid grid-cols-3 gap-1.5 sm:grid-cols-4">
              {PWA_ICON_OPTIONS.map((o) => {
                const selected = o.id === icon
                const thumb = brandingPreviewSvg(o.id, bg, fg || autoFg, 64)
                return (
                  <button
                    key={o.id}
                    type="button"
                    title={`${o.label} — ${o.hint}`}
                    onClick={() => setIcon(o.id)}
                    className={cn(
                      'relative flex flex-col items-center gap-1 rounded-xl border px-1.5 py-2 transition-colors',
                      selected ? 'border-primary bg-primary/10' : 'border-border/60 hover:bg-white/5',
                    )}
                  >
                    <img
                      src={`data:image/svg+xml;utf8,${encodeURIComponent(thumb)}`}
                      alt=""
                      className="h-10 w-10 rounded-lg"
                      width={40}
                      height={40}
                    />
                    <span className="text-[10px] font-medium">{o.label}</span>
                    {selected && <Check className="absolute top-1 right-1 h-3 w-3 text-primary" />}
                  </button>
                )
              })}
            </div>
          </div>

          <div>
            <p className="mb-1.5 text-[11px] font-medium text-muted-foreground">Warna latar</p>
            <div className="flex flex-wrap items-center gap-2">
              {PWA_BG_PRESETS.map((c) => (
                <button
                  key={c}
                  type="button"
                  title={c}
                  onClick={() => {
                    setBg(c)
                    setFg(/^#(f|e|d)/i.test(c) ? '#0c0d10' : '#ffffff')
                  }}
                  className={cn(
                    'h-7 w-7 rounded-full border-2 transition-transform',
                    bg.toLowerCase() === c.toLowerCase() ? 'scale-110 border-primary' : 'border-border/80',
                  )}
                  style={{ backgroundColor: c }}
                />
              ))}
              <label className="flex items-center gap-1.5 text-[11px] text-muted-foreground">
                Custom
                <input
                  type="color"
                  value={bg}
                  onChange={(e) => {
                    const c = e.target.value
                    setBg(c)
                    setFg(/^#(f|e|d)/i.test(c) ? '#0c0d10' : '#ffffff')
                  }}
                  className="h-7 w-9 cursor-pointer rounded border border-border bg-transparent"
                />
              </label>
            </div>
          </div>

          <div className="flex flex-wrap gap-2 pt-1">
            <Button onClick={() => void onSave()} disabled={saving || loading || !dirty}>
              {saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Check className="h-4 w-4" />}
              Simpan ikon
            </Button>
            <Button variant="outline" onClick={() => void onReset()} disabled={saving || loading || !saved?.custom}>
              <RotateCcw className="h-4 w-4" />
              Bawaan
            </Button>
          </div>
        </div>
      </div>
    </Section>
  )
}
