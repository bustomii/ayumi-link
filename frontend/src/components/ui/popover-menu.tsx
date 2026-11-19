import { useEffect, useLayoutEffect, useRef, useState, type ReactNode } from 'react'
import { createPortal } from 'react-dom'

type Props = {
  /** Elemen pemicu; menu diposisikan relatif terhadapnya. */
  anchor: HTMLElement
  onClose: () => void
  children: ReactNode
  width?: number
}

const GAP = 6
const MARGIN = 8

function readSafePx(name: '--safe-t' | '--safe-b' | '--safe-l' | '--safe-r') {
  const raw = getComputedStyle(document.documentElement).getPropertyValue(name).trim()
  const n = Number.parseFloat(raw)
  return Number.isFinite(n) ? n : 0
}

function viewportBox() {
  const vv = window.visualViewport
  const top = vv?.offsetTop ?? 0
  const left = vv?.offsetLeft ?? 0
  const height = vv?.height ?? window.innerHeight
  const width = vv?.width ?? window.innerWidth
  const safeT = readSafePx('--safe-t')
  const safeB = readSafePx('--safe-b')
  const safeL = readSafePx('--safe-l')
  const safeR = readSafePx('--safe-r')
  return {
    minTop: top + safeT + MARGIN,
    maxBottom: top + height - safeB - MARGIN,
    minLeft: left + safeL + MARGIN,
    maxRight: left + width - safeR - MARGIN,
  }
}

/**
 * Menu melayang di atas semua elemen (portal + posisi fixed), jadi tidak tertutup header yang
 * menempel atau terpotong area scroll. Membuka ke bawah, berbalik ke atas bila ruang kurang.
 * Di iOS PWA memakai visualViewport + safe-area agar tidak ketutup notch/home indicator.
 */
export function PopoverMenu({ anchor, onClose, children, width = 208 }: Props) {
  const ref = useRef<HTMLDivElement>(null)
  const [pos, setPos] = useState<{ top: number; left: number; maxHeight: number } | null>(null)

  useLayoutEffect(() => {
    const el = ref.current
    if (!el) return
    const a = anchor.getBoundingClientRect()
    const box = viewportBox()
    const avail = Math.max(120, box.maxBottom - box.minTop)
    el.style.maxHeight = `${avail}px`
    const h = el.offsetHeight

    let top = a.bottom + GAP
    if (top + h > box.maxBottom) top = Math.max(box.minTop, a.top - h - GAP)
    top = Math.min(Math.max(top, box.minTop), Math.max(box.minTop, box.maxBottom - Math.min(h, avail)))

    const left = Math.min(Math.max(box.minLeft, a.right - width), box.maxRight - width)
    setPos({ top, left, maxHeight: avail })
  }, [anchor, width, children])

  useEffect(() => {
    const onDown = (e: MouseEvent) => {
      const t = e.target as Node
      if (!ref.current?.contains(t) && !anchor.contains(t)) onClose()
    }
    const onKey = (e: KeyboardEvent) => e.key === 'Escape' && onClose()
    // Posisi menjadi usang bila daftar di-scroll, keyboard, atau jendela berubah ukuran.
    const onMove = () => onClose()
    document.addEventListener('mousedown', onDown)
    document.addEventListener('keydown', onKey)
    window.addEventListener('scroll', onMove, true)
    window.addEventListener('resize', onMove)
    window.visualViewport?.addEventListener('resize', onMove)
    window.visualViewport?.addEventListener('scroll', onMove)
    return () => {
      document.removeEventListener('mousedown', onDown)
      document.removeEventListener('keydown', onKey)
      window.removeEventListener('scroll', onMove, true)
      window.removeEventListener('resize', onMove)
      window.visualViewport?.removeEventListener('resize', onMove)
      window.visualViewport?.removeEventListener('scroll', onMove)
    }
  }, [anchor, onClose])

  return createPortal(
    <div
      ref={ref}
      role="menu"
      style={{
        position: 'fixed',
        top: pos?.top ?? -9999,
        left: pos?.left ?? -9999,
        width,
        maxHeight: pos?.maxHeight,
        overflowY: 'auto',
        WebkitOverflowScrolling: 'touch',
      }}
      className="z-[80] rounded-xl border border-border bg-card p-1 shadow-xl"
    >
      {children}
    </div>,
    document.body,
  )
}
