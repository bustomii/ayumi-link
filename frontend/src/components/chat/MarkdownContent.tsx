import type { Components } from 'react-markdown'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { CodeBlock, InlineCode } from '@/components/chat/CodeBlock'

const components: Components = {
  pre({ children }) {
    return <>{children}</>
  },
  code({ className, children }) {
    const text = String(children)
    const match = /language-([\w-]+)/.exec(className || '')
    const isBlock = Boolean(match) || (typeof children === 'string' && children.includes('\n'))

    if (isBlock) {
      return <CodeBlock language={match?.[1] || 'text'}>{text}</CodeBlock>
    }

    return <InlineCode className={className}>{children}</InlineCode>
  },
  a({ href, children }) {
    return (
      <a href={href} target="_blank" rel="noreferrer noopener">
        {children}
      </a>
    )
  },
  table({ children }) {
    return (
      <div className="chat-table-wrap">
        <table>{children}</table>
      </div>
    )
  },
  thead({ children }) {
    return <thead>{children}</thead>
  },
  tbody({ children }) {
    return <tbody>{children}</tbody>
  },
  tr({ children }) {
    return <tr>{children}</tr>
  },
  th({ children }) {
    return <th>{children}</th>
  },
  td({ children }) {
    return <td>{children}</td>
  },
}

export function MarkdownContent({ content }: { content: string }) {
  return (
    <div className="chat-md min-w-0 max-w-full overflow-hidden">
      <ReactMarkdown remarkPlugins={[remarkGfm]} components={components}>
        {content}
      </ReactMarkdown>
    </div>
  )
}
