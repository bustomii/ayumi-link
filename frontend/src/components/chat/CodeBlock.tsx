import { Check, Copy } from 'lucide-react'
import { useState, type ReactNode } from 'react'
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter'
import { oneDark } from 'react-syntax-highlighter/dist/esm/styles/prism'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

type Props = {
  language?: string
  children: string
  className?: string
}

const LANG_ALIASES: Record<string, string> = {
  sh: 'bash',
  shell: 'bash',
  zsh: 'bash',
  js: 'javascript',
  ts: 'typescript',
  py: 'python',
  md: 'markdown',
  yml: 'yaml',
}

export function CodeBlock({ language = '', children, className }: Props) {
  const [copied, setCopied] = useState(false)
  const raw = children.replace(/\n$/, '')
  const lang = LANG_ALIASES[language.toLowerCase()] || language.toLowerCase() || 'text'

  async function copy() {
    try {
      await navigator.clipboard.writeText(raw)
      setCopied(true)
      window.setTimeout(() => setCopied(false), 1500)
    } catch {
      /* ignore */
    }
  }

  return (
    <div
      className={cn(
        'group relative my-2 max-w-full overflow-hidden rounded-xl border border-border/80 bg-[#1e1e1e]',
        className,
      )}
    >
      <div className="flex items-center justify-between gap-2 border-b border-white/5 px-3 py-1.5">
        <span className="truncate text-[10px] font-medium uppercase tracking-wide text-white/45">
          {lang === 'text' ? 'code' : lang}
        </span>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="h-7 w-7 text-white/55 hover:bg-white/10 hover:text-white"
          onClick={() => void copy()}
          title="Copy"
        >
          {copied ? <Check className="h-3.5 w-3.5 text-emerald-400" /> : <Copy className="h-3.5 w-3.5" />}
        </Button>
      </div>
      <div className="max-w-full overflow-x-auto [-webkit-overflow-scrolling:touch]">
        <SyntaxHighlighter
          language={lang}
          style={oneDark}
          PreTag="div"
          customStyle={{
            margin: 0,
            padding: '0.85rem 1rem',
            background: 'transparent',
            fontSize: '0.78rem',
            lineHeight: 1.55,
          }}
          codeTagProps={{
            style: {
              fontFamily:
                'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace',
            },
          }}
          wrapLongLines={false}
        >
          {raw}
        </SyntaxHighlighter>
      </div>
    </div>
  )
}

export function InlineCode({
  children,
  className,
}: {
  children: ReactNode
  className?: string
}) {
  return (
    <code
      className={cn(
        'rounded-md bg-white/10 px-1.5 py-0.5 font-mono text-[0.82em] text-[#ce9178]',
        className,
      )}
    >
      {children}
    </code>
  )
}
