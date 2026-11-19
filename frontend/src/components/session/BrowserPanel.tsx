import { ArrowLeft, ArrowRight, CornerDownLeft, Delete, Globe, Loader2, Power, RotateCw } from 'lucide-react'
import { useCallback, useEffect, useRef, useState } from 'react'
import {
  closeBrowser,
  fetchBrowserFrame,
  fetchBrowserStatus,
  openBrowser,
  sendBrowserInput,
  type BrowserInput,
} from '@/api/browser'
import { ApiError } from '@/api/http'
import { Button } from '@/components/ui/button'

const FRAME_MS = 700

/** Pembulatan agar ukuran viewport tidak berubah-ubah tiap beberapa piksel. */
const snap = (n: number) => Math.max(200, Math.round(n / 10) * 10)

export function BrowserPanel() {
  const [available, setAvailable] = useState<boolean | null>(null)
  const [running, setRunning] = useState(false)
  const [address, setAddress] = useState('')
  const [title, setTitle] = useState('')
  const [typed, setTyped] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [src, setSrc] = useState<string | null>(null)

  const boxRef = useRef<HTMLDivElement>(null)
  const imgRef = useRef<HTMLImageElement>(null)
  const sizeRef = useRef({ w: 390, h: 700 })
  const editingAddress = useRef(false)
  const touchY = useRef<number | null>(null)

  // Ukuran area tampilan = ukuran layar virtual browser.
  useEffect(() => {
    const el = boxRef.current
    if (!el) return
    const measure = () => {
      const r = el.getBoundingClientRect()
      if (r.width > 0 && r.height > 0) sizeRef.current = { w: snap(r.width), h: snap(r.height) }
    }
    measure()
    const ro = new ResizeObserver(measure)
    ro.observe(el)
    return () => ro.disconnect()
  }, [])

  // Status awal.
  useEffect(() => {
    let off = false
    fetchBrowserStatus()
      .then((s) => {
        if (off) return
        setAvailable(s.available)
        setRunning(s.running)
        if (s.url) setAddress(s.url)
        if (s.title) setTitle(s.title)
      })
      .catch((e) => {
        if (!off) {
          setAvailable(false)
          setError(e instanceof Error ? e.message : 'Gagal membaca status browser')
        }
      })
    return () => {
      off = true
    }
  }, [])

  // Ambil frame berkala selama browser hidup dan tab terlihat.
  useEffect(() => {
    if (!running) return
    let off = false
    let timer: number | undefined
    const ctrl = new AbortController()
    const tick = async () => {
      if (document.hidden) {
        timer = window.setTimeout(tick, FRAME_MS)
        return
      }
      try {
        const f = await fetchBrowserFrame(sizeRef.current.w, sizeRef.current.h, ctrl.signal)
        if (off) return
        setSrc((old) => {
          if (old) URL.revokeObjectURL(old)
          return URL.createObjectURL(f.blob)
        })
        setTitle(f.title)
        if (!editingAddress.current) setAddress(f.url)
        setError(null)
      } catch (e) {
        if (off) return
        if (e instanceof ApiError && e.status === 409) {
          setRunning(false) // browser dimatikan (idle / ditutup dari tempat lain)
          return
        }
        if (!(e instanceof DOMException && e.name === 'AbortError')) {
          setError(e instanceof Error ? e.message : 'Gagal mengambil tampilan browser')
        }
      }
      if (!off) timer = window.setTimeout(tick, FRAME_MS)
    }
    void tick()
    return () => {
      off = true
      ctrl.abort()
      if (timer) window.clearTimeout(timer)
    }
  }, [running])

  useEffect(
    () => () => {
      setSrc((old) => {
        if (old) URL.revokeObjectURL(old)
        return null
      })
    },
    [],
  )

  const go = async (url: string) => {
    setBusy(true)
    setError(null)
    try {
      const s = await openBrowser(url, sizeRef.current.w, sizeRef.current.h)
      setRunning(s.running)
      if (s.url) setAddress(s.url)
      setTitle(s.title ?? '')
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Gagal membuka halaman')
    } finally {
      setBusy(false)
    }
  }

  const act = useCallback(async (input: BrowserInput) => {
    try {
      await sendBrowserInput(input)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Aksi gagal')
    }
  }, [])

  const toPoint = (clientX: number, clientY: number) => {
    const img = imgRef.current
    if (!img) return null
    const r = img.getBoundingClientRect()
    if (r.width === 0 || r.height === 0) return null
    return {
      x: ((clientX - r.left) * (img.naturalWidth || sizeRef.current.w)) / r.width,
      y: ((clientY - r.top) * (img.naturalHeight || sizeRef.current.h)) / r.height,
    }
  }

  const stop = async () => {
    setBusy(true)
    try {
      await closeBrowser()
      setRunning(false)
      setSrc(null)
      setTitle('')
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Gagal menutup browser')
    } finally {
      setBusy(false)
    }
  }

  if (available === false) {
    return (
      <div className="flex min-h-0 flex-1 flex-col items-center justify-center gap-2 bg-[#0a0b0e] p-6 text-center">
        <Globe className="h-8 w-8 text-muted-foreground" />
        <p className="text-sm text-foreground">Browser tidak tersedia</p>
        <p className="max-w-sm text-xs text-muted-foreground">
          {error ?? 'Chromium/Chrome tidak ditemukan di server. Docker: bangun ulang image. Native: pasang Chrome atau set AYUMI_BROWSER_BIN.'}
        </p>
      </div>
    )
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col overflow-hidden bg-[#0a0b0e]">
      <form
        className="flex shrink-0 items-center gap-1.5 border-b border-border/50 px-2 py-2"
        onSubmit={(e) => {
          e.preventDefault()
          editingAddress.current = false
          if (address.trim()) void go(address)
        }}
      >
        <Button type="button" variant="ghost" size="icon" className="h-8 w-8 shrink-0" disabled={!running} onClick={() => void act({ type: 'back' })} aria-label="Kembali">
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <Button type="button" variant="ghost" size="icon" className="h-8 w-8 shrink-0" disabled={!running} onClick={() => void act({ type: 'forward' })} aria-label="Maju">
          <ArrowRight className="h-4 w-4" />
        </Button>
        <Button type="button" variant="ghost" size="icon" className="h-8 w-8 shrink-0" disabled={!running} onClick={() => void act({ type: 'reload' })} aria-label="Muat ulang">
          <RotateCw className="h-4 w-4" />
        </Button>
        <input
          value={address}
          onChange={(e) => setAddress(e.target.value)}
          onFocus={() => (editingAddress.current = true)}
          onBlur={() => (editingAddress.current = false)}
          placeholder="Ketik alamat, mis. example.com"
          inputMode="url"
          autoCapitalize="off"
          autoCorrect="off"
          spellCheck={false}
          className="min-w-0 flex-1 rounded-lg border border-border bg-card/70 px-3 py-1.5 text-sm outline-none focus:border-primary"
        />
        <Button type="submit" size="sm" disabled={busy || !address.trim()} className="shrink-0">
          {busy ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : 'Buka'}
        </Button>
        <Button type="button" variant="ghost" size="icon" className="h-8 w-8 shrink-0" disabled={!running || busy} onClick={() => void stop()} aria-label="Matikan browser" title="Matikan browser (cookie tetap tersimpan)">
          <Power className="h-4 w-4" />
        </Button>
      </form>

      <div ref={boxRef} className="relative min-h-0 flex-1 overflow-hidden bg-black">
        {src && running ? (
          <img
            ref={imgRef}
            src={src}
            alt={title || 'Tampilan browser'}
            draggable={false}
            className="absolute left-0 top-0 block h-full w-full select-none object-fill"
            onClick={(e) => {
              const p = toPoint(e.clientX, e.clientY)
              if (p) void act({ type: 'click', ...p })
            }}
            onWheel={(e) => {
              const p = toPoint(e.clientX, e.clientY)
              if (p) void act({ type: 'scroll', ...p, deltaY: e.deltaY })
            }}
            onTouchStart={(e) => {
              touchY.current = e.touches[0]?.clientY ?? null
            }}
            onTouchMove={(e) => {
              const t = e.touches[0]
              if (!t || touchY.current === null) return
              const dy = touchY.current - t.clientY
              if (Math.abs(dy) < 24) return // gerakan kecil dianggap ketukan, bukan gulir
              touchY.current = t.clientY
              const p = toPoint(t.clientX, t.clientY)
              if (p) void act({ type: 'scroll', ...p, deltaY: dy * 2 })
            }}
            onTouchEnd={() => {
              touchY.current = null
            }}
          />
        ) : (
          <div className="flex h-full flex-col items-center justify-center gap-2 p-6 text-center">
            {available === null || busy ? (
              <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
            ) : (
              <>
                <Globe className="h-8 w-8 text-muted-foreground" />
                <p className="text-sm text-foreground">Browser di server</p>
                <p className="max-w-xs text-xs text-muted-foreground">
                  Chromium berjalan di mesin tempat Ayumi Link dipasang. Login situs tersimpan (cookie) dan bertahan
                  setelah browser dimatikan. Browser mati otomatis bila lama tidak dipakai.
                </p>
              </>
            )}
          </div>
        )}
      </div>

      {error && <p className="shrink-0 px-3 py-1.5 text-xs text-red-400">{error}</p>}

      {running && (
        <form
          className="flex shrink-0 items-center gap-1.5 border-t border-border/70 bg-background/40 px-2 py-2"
          onSubmit={(e) => {
            e.preventDefault()
            if (!typed) return
            void act({ type: 'type', text: typed })
            setTyped('')
          }}
        >
          <input
            value={typed}
            onChange={(e) => setTyped(e.target.value)}
            placeholder="Ketik ke kolom yang dipilih…"
            autoCapitalize="off"
            autoCorrect="off"
            spellCheck={false}
            className="min-w-0 flex-1 rounded-lg border border-border bg-card/70 px-3 py-1.5 text-sm outline-none focus:border-primary"
          />
          <Button type="submit" size="sm" variant="outline" disabled={!typed} className="shrink-0">
            Kirim
          </Button>
          <Button type="button" variant="ghost" size="icon" className="h-8 w-8 shrink-0" onClick={() => void act({ type: 'key', key: 'Backspace' })} aria-label="Backspace">
            <Delete className="h-4 w-4" />
          </Button>
          <Button type="button" variant="ghost" size="icon" className="h-8 w-8 shrink-0" onClick={() => void act({ type: 'key', key: 'Enter' })} aria-label="Enter">
            <CornerDownLeft className="h-4 w-4" />
          </Button>
        </form>
      )}
    </div>
  )
}
