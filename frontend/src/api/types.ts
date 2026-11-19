export type Session = {
  id: string
  title: string
  cwd: string
  project: string
  group?: string
  groupOrder?: number
  updatedAt: string
  live: boolean
  status?: string
  /** Dibuat lewat Ayumi Link (bisa dihapus permanen). */
  local?: boolean
  /** Folder kerja sudah tidak ada / tidak diizinkan: perlu fork ke folder lain. */
  cwdMissing?: boolean
}

export type AttachmentKind = 'image' | 'pdf' | 'excel' | 'word'

/** Lampiran pada sebuah pesan (id kosong bila tidak bisa dipetakan ke file di server). */
export type Attachment = {
  id: string
  name: string
  kind: AttachmentKind | string
}

export type Message = {
  id: string
  role: string
  content: string
  tools?: string[]
  timestamp: string
  attachments?: Attachment[]
}

export type ContextUsage = { used: number; window: number; model?: string }

export type SessionDetail = {
  session: Session
  messages: Message[]
  hasMore: boolean
  total: number
  limit: number
  context?: ContextUsage
}

export type DirListing = {
  path: string
  parent: string
  dirs: { name: string; path: string }[]
  truncated: boolean
}

export type FileChange = {
  path: string
  absPath: string
  backupName?: string
  version: number
  updatedAt?: string
  status: 'modified' | 'added' | 'deleted' | 'unknown' | string
}

export type ChangeDiff = FileChange & {
  before: string
  after: string
  diff: string
}

export type TermResult = {
  command: string
  cwd: string
  stdout: string
  stderr: string
  exitCode: number
  timedOut: boolean
  truncated: boolean
}

export type AuthStatus =
  | { state: 'setup_required' }
  | { state: 'logged_out' }
  | { state: 'authenticated'; username: string }
