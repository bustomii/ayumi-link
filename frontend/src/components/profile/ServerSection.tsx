import { Server } from 'lucide-react'
import type { ConfigInfo } from '@/api/profile'
import { InfoRow, Section } from '@/components/profile/Section'

const onOff = (v: boolean) => (v ? 'Aktif' : 'Nonaktif')

/** Konfigurasi server (hanya baca); diubah lewat environment saat menjalankan backend. */
export function ServerSection({ config }: { config: ConfigInfo }) {
  return (
    <Section
      icon={Server}
      tone="text-amber-400"
      title="Server"
      description="Hanya baca. Diatur lewat environment saat backend dijalankan."
    >
      <dl className="divide-y divide-border/50">
        <InfoRow label="Versi Ayumi Link">{config.version}</InfoRow>
        <InfoRow label="Alamat">
          <code className="font-mono text-xs">{config.address}</code>
        </InfoRow>
        <InfoRow label="CLI Claude">
          {config.claudeVersion || 'tidak terdeteksi'}
          <span className="mt-0.5 block break-all font-mono text-[11px] text-muted-foreground">{config.claudeBin}</span>
        </InfoRow>
        <InfoRow label="Folder yang diizinkan">
          {config.roots.length ? (
            <ul className="space-y-0.5">
              {config.roots.map((r) => (
                <li key={r} className="break-all font-mono text-xs">
                  {r}
                </li>
              ))}
            </ul>
          ) : (
            'tidak ada'
          )}
        </InfoRow>
        <InfoRow label="Izin tool bawaan server">{config.permissionMode}</InfoRow>
        <InfoRow label="Cookie Secure">{onOff(config.cookieSecure)}</InfoRow>
        <InfoRow label="Percaya proxy">{onOff(config.trustProxy)}</InfoRow>
        <InfoRow label="Folder data">
          <span className="break-all font-mono text-xs">{config.dataDir}</span>
        </InfoRow>
      </dl>
    </Section>
  )
}
