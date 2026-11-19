import { useEffect } from 'react'
import { isIos, isIosPwa } from '@/lib/pwa'

/**
 * Menyelaraskan tinggi shell ke area yang benar-benar terlihat (visualViewport).
 *
 * Tinggi ditulis ke --app-height / --app-top dan dipakai .app-shell (position: fixed). Pendekatan
 * yang sama dipakai di Android dan iOS; khusus iOS ditambah:
 *  - saat keyboard menutup / input kehilangan fokus, scroll halaman dikembalikan ke 0
 *    (iOS menggeser viewport saat fokus ke input, sehingga layout terlihat terpotong);
 *  - tinggi ikut berubah saat keyboard muncul, sehingga kolom chat tidak tertutup.
 */
export function usePwaViewport() {
  useEffect(() => {
    const root = document.documentElement
    const ios = isIos()
    root.classList.toggle('ios-pwa', isIosPwa())

    const sync = () => {
      const vv = window.visualViewport
      const height = Math.round(vv?.height ?? window.innerHeight)
      const top = Math.round(vv?.offsetTop ?? 0)
      root.style.setProperty('--app-height', `${height}px`)
      root.style.setProperty('--app-top', `${top}px`)
      // Keyboard terbuka: visualViewport jauh lebih pendek dari layar penuh.
      const kb = window.innerHeight - height > 120
      root.classList.toggle('kb-open', kb)
    }

    // iOS menggeser dokumen saat input fokus; shell fixed tidak boleh ikut tergeser.
    const resetScroll = () => {
      if (!ios) return
      if (window.scrollX !== 0 || window.scrollY !== 0) window.scrollTo(0, 0)
      if (document.scrollingElement && document.scrollingElement.scrollTop !== 0) {
        document.scrollingElement.scrollTop = 0
      }
    }
    const onFocusOut = () => {
      // Tunggu keyboard turun, lalu hitung ulang ukuran dan kembalikan posisi.
      window.setTimeout(() => {
        resetScroll()
        sync()
      }, 120)
      window.setTimeout(() => {
        resetScroll()
        sync()
      }, 450)
    }
    const onVvChange = () => {
      sync()
      resetScroll()
    }

    sync()
    const vv = window.visualViewport
    vv?.addEventListener('resize', onVvChange)
    vv?.addEventListener('scroll', onVvChange)
    window.addEventListener('resize', onVvChange)
    window.addEventListener('orientationchange', onVvChange)
    window.addEventListener('pageshow', onVvChange)
    document.addEventListener('visibilitychange', onVvChange)
    document.addEventListener('focusout', onFocusOut)
    window.addEventListener('scroll', resetScroll, { passive: true })
    return () => {
      vv?.removeEventListener('resize', onVvChange)
      vv?.removeEventListener('scroll', onVvChange)
      window.removeEventListener('resize', onVvChange)
      window.removeEventListener('orientationchange', onVvChange)
      window.removeEventListener('pageshow', onVvChange)
      document.removeEventListener('visibilitychange', onVvChange)
      document.removeEventListener('focusout', onFocusOut)
      window.removeEventListener('scroll', resetScroll)
      root.classList.remove('ios-pwa', 'kb-open')
    }
  }, [])
}
