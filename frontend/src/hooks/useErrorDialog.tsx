import { useCallback, useRef, useState, type ReactNode } from 'react'
import { ErrorDialog } from '@/components/ui/error-dialog'
import { describeError, type AppError } from '@/lib/errors'
import { serverReachable } from '@/lib/reachability'

type Shown = { error: AppError; onRetry?: () => void }

/**
 * Menampilkan error sebagai popup. `show(e, aksi)` menerjemahkan error apa pun ke pesan yang jelas;
 * render `dialog` sekali di komponen induk.
 *
 * Error jaringan diperiksa dulu: bila server ternyata masih terjangkau (mis. request terputus saat
 * halaman dibuka ulang), popup "tidak bisa terhubung" tidak ditampilkan.
 */
export function useErrorDialog(): {
  show: (e: unknown, action?: string, onRetry?: () => void) => void
  /** Tampilkan error yang sudah disusun sendiri (mis. daftar file yang ditolak). */
  showApp: (error: AppError, onRetry?: () => void) => void
  dialog: ReactNode
} {
  const [shown, setShown] = useState<Shown | null>(null)
  const checking = useRef(false)
  const lastAutoRetry = useRef(0)

  // Popup yang sedang terbuka tidak ditimpa error berikutnya (biasanya penyebabnya sama).
  const open = useCallback((error: AppError, onRetry?: () => void) => {
    setShown((prev) => prev ?? { error, onRetry })
  }, [])

  const show = useCallback(
    (e: unknown, action?: string, onRetry?: () => void) => {
      const error = describeError(e, action)
      if (error.kind !== 'network') {
        open(error, onRetry)
        return
      }
      if (checking.current) return // pemeriksaan untuk error jaringan lain sedang berjalan
      checking.current = true
      void serverReachable()
        .then((up) => {
          if (!up) {
            open(error, onRetry) // memang tidak terjangkau
          } else if (onRetry && Date.now() - lastAutoRetry.current > 15_000) {
            lastAutoRetry.current = Date.now() // maksimal satu kali per 15 detik, hindari putaran tanpa akhir
            onRetry() // kegagalan sesaat; ulangi diam-diam
          } else {
            // Server hidup tetapi permintaan ini terputus. Jangan klaim server mati.
            open({
              kind: 'network',
              title: 'Permintaan terputus',
              message: `${action ?? 'Permintaan'} sempat terputus, tetapi server dapat dijangkau.`,
              hint: 'Ulangi aksi tersebut.',
              detail: error.detail,
            })
          }
        })
        .finally(() => {
          checking.current = false
        })
    },
    [open],
  )

  const showApp = useCallback((error: AppError, onRetry?: () => void) => open(error, onRetry), [open])

  const dialog = shown ? (
    <ErrorDialog error={shown.error} onRetry={shown.onRetry} onClose={() => setShown(null)} />
  ) : null

  return { show, showApp, dialog }
}
