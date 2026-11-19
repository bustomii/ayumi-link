import { AuthGate } from '@/components/auth/AuthGate'
import { InstallBanner } from '@/components/pwa/InstallBanner'
import { PullReloadIndicator } from '@/components/pwa/PullReloadIndicator'
import { usePwaPullReload } from '@/hooks/usePwaPullReload'
import { usePwaViewport } from '@/hooks/usePwaViewport'
import { Workspace } from '@/Workspace'

export default function App() {
  usePwaViewport()
  const pull = usePwaPullReload(true)

  return (
    <>
      <AuthGate>{({ username, logout }) => <Workspace username={username} onLogout={logout} />}</AuthGate>
      <InstallBanner />
      <PullReloadIndicator {...pull} />
    </>
  )
}
