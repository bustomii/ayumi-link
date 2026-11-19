import { Loader2, Play, Terminal } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { enterTerminal, runTerminal } from '@/api/sessions'
import type { TermResult } from '@/api/types'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'

type Props = { sessionId?: string }

export function TerminalPanel({ sessionId }: Props) {
  const [cwd, setCwd] = useState('')
  const [lines, setLines] = useState<TermResult[]>([])
  const [cmd, setCmd] = useState('')
  const [loading, setLoading] = useState(true)
  const [running, setRunning] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const bottomRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError(null)
    void enterTerminal(sessionId || undefined)
      .then((d) => {
        if (!cancelled) setCwd(d.cwd ?? '')
      })
      .catch((e) => {
        if (!cancelled) setError(e instanceof Error ? e.message : 'Gagal membuka terminal')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [sessionId])

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ block: 'end' })
  }, [lines, running])

  const run = async () => {
    const command = cmd.trim()
    if (!command || running) return
    setRunning(true)
    setError(null)
    try {
      const result = await runTerminal(command)
      setLines((r) => [...r, result])
      setCwd(result.cwd)
      setCmd('')
      inputRef.current?.focus()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Gagal menjalankan perintah')
    } finally {
      setRunning(false)
    }
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col overflow-hidden bg-[#0a0b0e]">
      <div className="flex shrink-0 items-center gap-2 border-b border-border/50 px-3 py-2">
        <Terminal className="h-4 w-4 text-emerald-400" />
        <p className="min-w-0 flex-1 truncate font-mono text-[11px] text-muted-foreground" title={cwd}>
          {loading ? '…' : cwd || 'cwd'}
        </p>
      </div>

      <ScrollArea className="min-h-0 flex-1 overflow-y-auto p-3 font-mono text-[11px] leading-relaxed sm:text-xs">
        {lines.length === 0 && !loading && (
          <p className="text-muted-foreground">
            Shell interaktif (satu sesi). Bisa <code className="text-foreground">cd</code> seperti terminal biasa.
            Folder awal = folder chat ini, atau root <code className="text-foreground">AYUMI_ROOTS</code>.
          </p>
        )}
        {lines.map((r, i) => (
          <div key={i} className="mb-3 border-b border-white/5 pb-3">
            <p className="text-sky-300">
              <span className="text-muted-foreground">$ </span>
              {r.command}
            </p>
            {r.stdout && (
              <pre className="mt-1 whitespace-pre-wrap break-words text-foreground/85">{r.stdout}</pre>
            )}
            {r.stderr && (
              <pre className="mt-1 whitespace-pre-wrap break-words text-red-300">{r.stderr}</pre>
            )}
            <p className="mt-1 text-[10px] text-muted-foreground">
              exit {r.exitCode}
              {r.timedOut ? ' · timeout' : ''}
              {r.truncated ? ' · output dipotong' : ''}
              {' · '}
              {r.cwd}
            </p>
          </div>
        ))}
        <div ref={bottomRef} />
      </ScrollArea>

      {error && <p className="shrink-0 px-3 py-1.5 text-xs text-red-400">{error}</p>}

      <form
        className="flex shrink-0 items-center gap-2 border-t border-border/70 bg-background/40 px-3 py-2"
        onSubmit={(e) => {
          e.preventDefault()
          void run()
        }}
      >
        <span className="font-mono text-sm text-emerald-400">$</span>
        <input
          ref={inputRef}
          value={cmd}
          onChange={(e) => setCmd(e.target.value)}
          disabled={running || loading}
          placeholder="cd folder && ls  (timeout 30s)"
          className="min-w-0 flex-1 rounded-lg border border-border bg-card/70 px-3 py-2 font-mono text-sm outline-none focus:border-primary disabled:opacity-60"
          autoCapitalize="off"
          autoCorrect="off"
          spellCheck={false}
        />
        <Button type="submit" size="sm" disabled={running || loading || !cmd.trim()} className="gap-1">
          {running ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Play className="h-3.5 w-3.5" />}
          Run
        </Button>
      </form>
    </div>
  )
}
