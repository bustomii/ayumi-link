import { useEffect } from 'react'
import { AuthGate } from '@/components/auth/AuthGate'
import { InstallBanner } from '@/components/pwa/InstallBanner'
import { PullReloadIndicator } from '@/components/pwa/PullReloadIndicator'
import { usePwaPullReload } from '@/hooks/usePwaPullReload'
import { usePwaViewport } from '@/hooks/usePwaViewport'
import { applyPwaBrandingLinks, loadPublicBranding } from '@/lib/pwa-branding'
import { Workspace } from '@/Workspace'

export default function App() {
  usePwaViewport()
  const pull = usePwaPullReload(true)

  useEffect(() => {
    void loadPublicBranding().then((b) => {
      if (b) applyPwaBrandingLinks(b)
    })
  }, [])

  return (
    <>
      <AuthGate>{({ username, logout }) => <Workspace username={username} onLogout={logout} />}</AuthGate>
      <InstallBanner />
      <PullReloadIndicator {...pull} />
    </>
  )
}
