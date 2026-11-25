import { Bot, User, Wrench } from 'lucide-react'
import type { Message } from '@/api/sessions'
import { ChatTag } from '@/components/chat/ChatTag'
import { MarkdownContent } from '@/components/chat/MarkdownContent'
import { MessageAttachments } from '@/components/chat/MessageAttachments'
import { emptyKnown, segmentMessage, type Known } from '@/lib/chat-tags'
import { cn } from '@/lib/utils'

function formatTime(iso: string) {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  return d.toLocaleString('id-ID', {
    day: '2-digit',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function isNoiseContent(content: string) {
  const t = content.trim()
  return /^\[tool:\s*.+\]$/i.test(t) || t === '📷 [gambar]'
}

function looksLikeMarkdown(content: string) {
  return (
    /```/.test(content) ||
    /(^|\n)\s*#{1,3}\s/.test(content) ||
    /(^|\n)\s*[-*]\s+\S/.test(content) ||
    /`[^`]+`/.test(content) ||
    /\*\*[^*]+\*\*/.test(content)
  )
}

export function ChatBubble({ message, known = emptyKnown }: { message: Message; known?: Known }) {
  const isUser = message.role === 'user'
  if (isNoiseContent(message.content)) return null

  const attachments = message.attachments ?? []
  const segments = isUser ? segmentMessage(message.content, known) : null
  const hasInline = !!segments && segments.some((x) => x.type !== 'text')
  const body = message.content
  const hasText = body.trim() !== ''
  const useMarkdown = !isUser || looksLikeMarkdown(body)

  return (
    <div
      className={cn(
        'flex w-full min-w-0 gap-2 sm:gap-3',
        isUser ? 'flex-row-reverse' : 'flex-row',
      )}
    >
      <div
        className={cn(
          'mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-full sm:h-9 sm:w-9',
          isUser
            ? 'bg-gradient-to-br from-sky-500 to-blue-700 text-white'
            : 'bg-gradient-to-br from-violet-500 to-fuchsia-600 text-white',
        )}
        title={isUser ? 'User' : 'Assistant'}
      >
        {isUser ? <User className="h-3.5 w-3.5 sm:h-4 sm:w-4" /> : <Bot className="h-3.5 w-3.5 sm:h-4 sm:w-4" />}
      </div>

      <div
        className={cn(
          'flex min-w-0 flex-1 flex-col gap-1',
          isUser ? 'items-end' : 'items-start',
        )}
      >
        <div className="flex max-w-full items-center gap-2 px-0.5">
          <span className="text-[11px] font-medium text-muted-foreground sm:text-xs">
            {isUser ? 'Kamu' : 'Claude'}
          </span>
          {message.timestamp && (
            <span className="text-[10px] text-muted-foreground/70">
              {formatTime(message.timestamp)}
            </span>
          )}
        </div>

        {!!message.tools?.length && (
          <div className="flex max-w-full flex-wrap gap-1 px-0.5">
            {message.tools.slice(0, 6).map((t) => (
              <span
                key={t}
                className="inline-flex max-w-[9rem] items-center gap-1 truncate rounded-full border border-border/80 bg-muted/60 px-2 py-0.5 text-[10px] text-muted-foreground"
              >
                <Wrench className="h-3 w-3 shrink-0" />
                <span className="truncate">{t}</span>
              </span>
            ))}
            {message.tools.length > 6 && (
              <span className="text-[10px] text-muted-foreground">
                +{message.tools.length - 6}
              </span>
            )}
          </div>
        )}

        <MessageAttachments attachments={attachments} isUser={isUser} />

        {hasText && (
          <div
            className={cn(
              'min-w-0 max-w-full overflow-hidden rounded-2xl px-3 py-2.5 text-sm leading-relaxed shadow-sm sm:max-w-[min(100%,40rem)] sm:px-4 sm:py-3',
              isUser
                ? 'rounded-tr-md bg-accent text-accent-foreground'
                : 'w-full rounded-tl-md border border-border/70 bg-card text-foreground',
            )}
          >
            {hasInline && segments ? (
              <p className="whitespace-pre-wrap break-words [overflow-wrap:anywhere]">
                {segments.map((seg, i) =>
                  seg.type === 'text' ? (
                    <span key={i}>{seg.text}</span>
                  ) : seg.type === 'folder' ? (
                    <ChatTag key={i} kind="folder" label={seg.name} title={seg.path} />
                  ) : (
                    <ChatTag key={i} kind={seg.type} label={seg.slug} />
                  ),
                )}
              </p>
            ) : useMarkdown ? (
              <MarkdownContent content={body} />
            ) : (
              <p className="whitespace-pre-wrap break-words [overflow-wrap:anywhere]">
                {body}
              </p>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
