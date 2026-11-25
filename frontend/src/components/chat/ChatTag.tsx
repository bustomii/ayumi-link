import { Bot, Folder, Wrench, X } from 'lucide-react'
import { cn } from '@/lib/utils'

export type TagKind = 'agent' | 'skill' | 'folder'

export const tagStyles: Record<TagKind, string> = {
  agent: 'bg-sky-200 text-sky-950',
  skill: 'bg-violet-200 text-violet-950',
  folder: 'bg-emerald-200 text-emerald-950',
}

export const tagPrefix: Record<TagKind, string> = { agent: '@', skill: '#', folder: '/' }

/** Badge berwarna per jenis: @agent (biru), #skill (ungu), /folder (hijau). */
export function ChatTag({
  kind,
  label,
  title,
  onRemove,
}: {
  kind: TagKind
  label: string
  title?: string
  onRemove?: () => void
}) {
  const Icon = kind === 'agent' ? Bot : kind === 'skill' ? Wrench : Folder
  return (
    <span
      title={title ?? `${tagPrefix[kind]}${label}`}
      className={cn(
        'inline-flex max-w-full items-center gap-1 rounded-full py-0.5 pl-2 text-xs font-medium',
        onRemove ? 'pr-1' : 'pr-2',
        tagStyles[kind],
      )}
    >
      <Icon className="h-3 w-3 shrink-0" />
      <span className="truncate">
        {tagPrefix[kind]}
        {label}
      </span>
      {onRemove && (
        <button
          type="button"
          aria-label={`Hapus ${tagPrefix[kind]}${label}`}
          onClick={onRemove}
          className="rounded-full p-0.5 hover:bg-black/15"
        >
          <X className="h-3 w-3" />
        </button>
      )}
    </span>
  )
}
