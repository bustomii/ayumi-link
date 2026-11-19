import type { ReactNode } from 'react'
import { AuthLayout } from '@/components/auth/AuthLayout'
import { AuthSkeleton } from '@/components/auth/AuthSkeleton'
import { LoginFlow } from '@/components/auth/LoginFlow'
import { SetupFlow } from '@/components/auth/SetupFlow'
import { Button } from '@/components/ui/button'
import { useAuth } from '@/hooks/useAuth'

type Props = { children: (ctx: { username: string; logout: () => void }) => ReactNode }

/** Menampilkan setup/login sampai pengguna terautentikasi, lalu merender aplikasi. */
export function AuthGate({ children }: Props) {
  const { state, refresh, logout, retry } = useAuth()

  switch (state.status) {
    case 'loading':
      return <AuthSkeleton />
    case 'error':
      return (
        <AuthLayout title="Tidak terhubung" description={state.message}>
          <p className="mb-4 text-sm text-muted-foreground">Pastikan backend Ayumi Link berjalan.</p>
          <Button className="w-full" onClick={retry}>
            Coba lagi
          </Button>
        </AuthLayout>
      )
    case 'setup_required':
      return <SetupFlow onDone={() => void refresh()} />
    case 'logged_out':
      return <LoginFlow onDone={() => void refresh()} />
    case 'authenticated':
      return <>{children({ username: state.username, logout: () => void logout() })}</>
  }
}
