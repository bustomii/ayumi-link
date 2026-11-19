import { useEffect, useRef, useState } from 'react'
import { isStandalonePwa } from '@/lib/pwa'

const THRESHOLD = 72
const MAX_PULL = 110

function canScroll(el: Element | null): boolean {
  if (!el || !(el instanceof HTMLElement)) return false
  return el.scrollHeight > el.clientHeight + 1
}

function findScrollParent(start: EventTarget | null): HTMLElement | null {
  let el = start instanceof HTMLElement ? start : null
  while (el && el !== document.body) {
    const style = getComputedStyle(el)
    const oy = style.overflowY
    if ((oy === 'auto' || oy === 'scroll' || oy === 'overlay') && canScroll(el)) {
      return el
    }
    el = el.parentElement
  }
  return null
}

export type PullReloadState = {
  /** Jarak tarik ke bawah (px), 0 = idle */
  distance: number
  /** Sudah melewati threshold — lepas jari = reload */
  armed: boolean
  /** Sedang reload */
  reloading: boolean
}

/**
 * Di PWA: tarik ke bawah saat tidak ada scroll → indikator + reload bila lepas di atas threshold.
 * Lepas lebih awal / batal → tidak reload.
 */
export function usePwaPullReload(enabled = true): PullReloadState {
  const startY = useRef(0)
  const pulling = useRef(false)
  const distanceRef = useRef(0)
  const [distance, setDistance] = useState(0)
  const [armed, setArmed] = useState(false)
  const [reloading, setReloading] = useState(false)

  useEffect(() => {
    if (!enabled || !isStandalonePwa()) return

    const reset = () => {
      pulling.current = false
      distanceRef.current = 0
      setDistance(0)
      setArmed(false)
    }

    const onStart = (e: TouchEvent) => {
      if (e.touches.length !== 1 || reloading) return
      const scrollParent = findScrollParent(e.target)
      if (scrollParent && canScroll(scrollParent) && scrollParent.scrollTop > 0) {
        reset()
        return
      }
      // Hanya gesture reload jika tidak ada area scroll (atau kosong di puncak tanpa overflow).
      if (scrollParent && canScroll(scrollParent)) {
        reset()
        return
      }
      startY.current = e.touches[0].clientY
      pulling.current = true
      distanceRef.current = 0
      setDistance(0)
      setArmed(false)
    }

    const onMove = (e: TouchEvent) => {
      if (!pulling.current || e.touches.length !== 1) return
      const dy = Math.max(0, e.touches[0].clientY - startY.current)
      if (dy > 8) e.preventDefault()
      const clamped = Math.min(MAX_PULL, dy)
      distanceRef.current = clamped
      setDistance(clamped)
      setArmed(clamped >= THRESHOLD)
    }

    const onEnd = () => {
      if (!pulling.current) return
      const dy = distanceRef.current
      const shouldReload = dy >= THRESHOLD
      pulling.current = false
      if (shouldReload) {
        setArmed(true)
        setReloading(true)
        setDistance(THRESHOLD)
        window.location.reload()
        return
      }
      reset()
    }

    const onCancel = () => {
      if (!pulling.current) return
      reset()
    }

    document.addEventListener('touchstart', onStart, { passive: true })
    document.addEventListener('touchmove', onMove, { passive: false })
    document.addEventListener('touchend', onEnd, { passive: true })
    document.addEventListener('touchcancel', onCancel, { passive: true })
    return () => {
      document.removeEventListener('touchstart', onStart)
      document.removeEventListener('touchmove', onMove)
      document.removeEventListener('touchend', onEnd)
      document.removeEventListener('touchcancel', onCancel)
    }
  }, [enabled, reloading])

  return { distance, armed, reloading }
}
