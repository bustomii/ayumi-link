import { useEffect } from 'react'
import { isIosPwa } from '@/lib/pwa'

/**
 * Menyelaraskan tinggi shell ke viewport.
 * iOS PWA: fixed inset 0 + safe-area CSS (hindari celah bawah / atas kepotong dari visualViewport).
 */
export function usePwaViewport() {
  useEffect(() => {
    const root = document.documentElement
    const iosPwa = isIosPwa()
    root.classList.toggle('ios-pwa', iosPwa)

    const sync = () => {
      if (iosPwa) {
        // Isi layar penuh fixed; padding notch/home lewat --safe-* (env).
        root.style.setProperty('--app-height', '100lvh')
        root.style.setProperty('--app-top', '0px')
        root.style.removeProperty('--safe-b')
        root.style.removeProperty('--safe-t')
        return
      }
      const vv = window.visualViewport
      const height = Math.round(vv?.height ?? window.innerHeight)
      const top = Math.round(vv?.offsetTop ?? 0)
      root.style.setProperty('--app-height', `${height}px`)
      root.style.setProperty('--app-top', `${top}px`)
    }

    sync()
    if (iosPwa) {
      window.addEventListener('resize', sync)
      window.addEventListener('orientationchange', sync)
      return () => {
        window.removeEventListener('resize', sync)
        window.removeEventListener('orientationchange', sync)
        root.classList.remove('ios-pwa')
        root.style.removeProperty('--app-height')
        root.style.removeProperty('--app-top')
      }
    }

    const vv = window.visualViewport
    vv?.addEventListener('resize', sync)
    vv?.addEventListener('scroll', sync)
    window.addEventListener('resize', sync)
    window.addEventListener('orientationchange', sync)
    return () => {
      vv?.removeEventListener('resize', sync)
      vv?.removeEventListener('scroll', sync)
      window.removeEventListener('resize', sync)
      window.removeEventListener('orientationchange', sync)
      root.classList.remove('ios-pwa')
    }
  }, [])
}
