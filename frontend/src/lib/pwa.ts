/** Deteksi PWA / iOS untuk perlakuan layout khusus. */

export function isStandalonePwa() {
  if (typeof window === 'undefined') return false
  return (
    window.matchMedia('(display-mode: standalone)').matches ||
    ('standalone' in navigator && Boolean((navigator as Navigator & { standalone?: boolean }).standalone))
  )
}

export function isIos() {
  if (typeof navigator === 'undefined') return false
  return (
    /iphone|ipad|ipod/i.test(navigator.userAgent) ||
    (navigator.platform === 'MacIntel' && navigator.maxTouchPoints > 1)
  )
}

export function isIosPwa() {
  return isIos() && isStandalonePwa()
}
