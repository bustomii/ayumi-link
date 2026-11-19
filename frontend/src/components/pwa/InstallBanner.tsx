import { useEffect, useState } from 'react'
import { Download, Share, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { isIos, isStandalonePwa } from '@/lib/pwa'

const DISMISS_KEY = 'ayumi-pwa-install-dismissed'

type BeforeInstallPromptEvent = Event & {
  prompt: () => Promise<void>
  userChoice: Promise<{ outcome: 'accepted' | 'dismissed' }>
}

/** Banner install PWA: tombol native di Android/Chrome, petunjuk Share di iOS Safari. */
export function InstallBanner() {
  const [deferred, setDeferred] = useState<BeforeInstallPromptEvent | null>(null)
  const [iosHint, setIosHint] = useState(false)
  const [hidden, setHidden] = useState(true)

  useEffect(() => {
    if (isStandalonePwa() || localStorage.getItem(DISMISS_KEY) === '1') return

    const onBip = (e: Event) => {
      e.preventDefault()
      setDeferred(e as BeforeInstallPromptEvent)
      setHidden(false)
    }
    window.addEventListener('beforeinstallprompt', onBip)

    if (isIos()) {
      setIosHint(true)
      setHidden(false)
    }

    return () => window.removeEventListener('beforeinstallprompt', onBip)
  }, [])

  if (hidden) return null

  const dismiss = () => {
    localStorage.setItem(DISMISS_KEY, '1')
    setHidden(true)
    setDeferred(null)
    setIosHint(false)
  }

  const install = async () => {
    if (!deferred) return
    await deferred.prompt()
    await deferred.userChoice
    dismiss()
  }

  return (
    <div className="pointer-events-none fixed inset-x-0 bottom-0 z-50 flex justify-center p-3 pb-[max(0.75rem,env(safe-area-inset-bottom))]">
      <div className="pointer-events-auto flex max-w-md items-start gap-3 rounded-xl border border-border bg-card/95 px-3 py-2.5 shadow-lg backdrop-blur">
        <div className="min-w-0 flex-1 text-sm">
          <p className="font-medium text-foreground">Pasang Ayumi Link</p>
          {deferred ? (
            <p className="mt-0.5 text-xs text-muted-foreground">Tambahkan ke layar utama seperti aplikasi.</p>
          ) : iosHint ? (
            <p className="mt-0.5 text-xs text-muted-foreground">
              Di Safari: ketuk <Share className="mx-0.5 inline h-3.5 w-3.5" aria-hidden /> lalu{' '}
              <span className="font-medium text-foreground">Add to Home Screen</span>.
            </p>
          ) : null}
        </div>
        {deferred ? (
          <Button type="button" size="sm" className="shrink-0 gap-1" onClick={() => void install()}>
            <Download className="h-3.5 w-3.5" aria-hidden />
            Pasang
          </Button>
        ) : null}
        <button
          type="button"
          className="shrink-0 rounded-md p-1 text-muted-foreground hover:bg-muted hover:text-foreground"
          aria-label="Tutup"
          onClick={dismiss}
        >
          <X className="h-4 w-4" />
        </button>
      </div>
    </div>
  )
}
