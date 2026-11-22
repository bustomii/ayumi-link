import { FitAddon } from '@xterm/addon-fit'
import { Terminal as XTerm } from '@xterm/xterm'
import '@xterm/xterm/css/xterm.css'
import { Loader2, Terminal } from 'lucide-react'
import { useCallback, useEffect, useRef, useState } from 'react'
import {
  createTerminalTab,
  deleteTerminalTab,
  fetchToolsTabs,
  terminalWsUrl,
  type ToolTab,
} from '@/api/toolsTabs'
import { ToolTabBar } from '@/components/session/ToolTabBar'

type Props = { sessionId: string }

/** Terminal PTY penuh (xterm) + multi-tab per chat. */
export function TerminalPanel({ sessionId }: Props) {
  const [tabs, setTabs] = useState<ToolTab[]>([])
  const [activeId, setActiveId] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  const hostRef = useRef<HTMLDivElement>(null)
  const xtermRef = useRef<XTerm | null>(null)
  const fitRef = useRef<FitAddon | null>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const activeRef = useRef(activeId)
  activeRef.current = activeId

  const reloadTabs = useCallback(async () => {
    const data = await fetchToolsTabs(sessionId)
    setTabs(data.terminal)
    return data.terminal
  }, [sessionId])

  // Muat / buat tab awal.
  useEffect(() => {
    let off = false
    setLoading(true)
    setError(null)
    ;(async () => {
      try {
        let list = await reloadTabs()
        if (off) return
        if (list.length === 0) {
          const t = await createTerminalTab(sessionId)
          if (off) return
          list = await reloadTabs()
          setActiveId(t.id)
        } else {
          const active = list.find((t) => t.active) ?? list[0]
          setActiveId(active.id)
        }
      } catch (e) {
        if (!off) setError(e instanceof Error ? e.message : 'Gagal memuat terminal')
      } finally {
        if (!off) setLoading(false)
      }
    })()
    return () => {
      off = true
    }
  }, [sessionId, reloadTabs])

  // xterm + websocket per tab aktif.
  useEffect(() => {
    if (!activeId || !hostRef.current) return
    const host = hostRef.current
    host.innerHTML = ''

    const term = new XTerm({
      cursorBlink: true,
      fontSize: 12,
      fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace',
      theme: { background: '#0a0b0e', foreground: '#e8e9ed', cursor: '#3d8bfd' },
      scrollback: 5000,
    })
    const fit = new FitAddon()
    term.loadAddon(fit)
    term.open(host)
    fit.fit()
    xtermRef.current = term
    fitRef.current = fit

    const ws = new WebSocket(terminalWsUrl(sessionId, activeId))
    ws.binaryType = 'arraybuffer'
    wsRef.current = ws

    ws.onopen = () => {
      const dims = { cols: term.cols, rows: term.rows }
      ws.send(JSON.stringify({ type: 'resize', cols: dims.cols, rows: dims.rows }))
    }
    ws.onmessage = (ev) => {
      if (typeof ev.data === 'string') {
        term.write(ev.data)
        return
      }
      term.write(new Uint8Array(ev.data as ArrayBuffer))
    }
    ws.onerror = () => setError('Koneksi terminal terputus')
    ws.onclose = () => {
      /* reconnect via tab switch / remount */
    }

    const sub = term.onData((data) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'input', data }))
      }
    })

    const onResize = () => {
      fit.fit()
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'resize', cols: term.cols, rows: term.rows }))
      }
    }
    const ro = new ResizeObserver(onResize)
    ro.observe(host)
    window.addEventListener('resize', onResize)
    // iOS: keyboard muncul/hilang mengubah visualViewport; hitung ulang kolom/baris setelah layout
    // selesai bergeser, lalu pastikan baris terakhir (prompt) terlihat.
    const vv = window.visualViewport
    let timers: number[] = []
    const refit = () => {
      timers.forEach((t) => window.clearTimeout(t))
      timers = [60, 250, 600].map((ms) =>
        window.setTimeout(() => {
          onResize()
          term.scrollToBottom()
        }, ms),
      )
    }
    vv?.addEventListener('resize', refit)

    return () => {
      timers.forEach((t) => window.clearTimeout(t))
      vv?.removeEventListener('resize', refit)
      sub.dispose()
      ro.disconnect()
      window.removeEventListener('resize', onResize)
      ws.close()
      term.dispose()
      xtermRef.current = null
      fitRef.current = null
      wsRef.current = null
    }
  }, [sessionId, activeId])

  const onNew = async () => {
    setBusy(true)
    setError(null)
    try {
      const t = await createTerminalTab(sessionId, `Terminal ${tabs.length + 1}`)
      await reloadTabs()
      setActiveId(t.id)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Gagal buat tab')
    } finally {
      setBusy(false)
    }
  }

  const onClose = async (id: string) => {
    setBusy(true)
    try {
      await deleteTerminalTab(sessionId, id)
      const list = await reloadTabs()
      if (activeId === id) {
        setActiveId(list[0]?.id ?? '')
      }
      if (list.length === 0) {
        const t = await createTerminalTab(sessionId)
        await reloadTabs()
        setActiveId(t.id)
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Gagal tutup tab')
    } finally {
      setBusy(false)
    }
  }

  const cwd = tabs.find((t) => t.id === activeId)?.cwd

  return (
    <div className="flex min-h-0 flex-1 flex-col overflow-hidden bg-[#0a0b0e] ios-tool-pad">
      <div className="flex shrink-0 items-center gap-2 border-b border-border/50 px-3 py-2">
        <Terminal className="h-4 w-4 text-emerald-400" />
        <p className="min-w-0 flex-1 truncate font-mono text-[11px] text-muted-foreground" title={cwd}>
          {loading ? '…' : cwd || 'terminal'}
        </p>
        {busy && <Loader2 className="h-3.5 w-3.5 animate-spin text-muted-foreground" />}
      </div>
      <ToolTabBar
        tabs={tabs}
        activeId={activeId}
        onSelect={setActiveId}
        onClose={(id) => void onClose(id)}
        onNew={() => void onNew()}
        busy={busy || loading}
      />
      {error && <p className="shrink-0 px-3 py-1.5 text-xs text-red-400">{error}</p>}
      <div ref={hostRef} className="min-h-0 flex-1 overflow-hidden px-2 pb-3 pt-2" />
    </div>
  )
}
