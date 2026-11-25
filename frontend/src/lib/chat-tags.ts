export type Known = { agents: Set<string>; skills: Set<string> }

export type Segment =
  | { type: 'text'; text: string }
  | { type: 'agent' | 'skill'; slug: string }
  | { type: 'folder'; path: string; name: string }

export const emptyKnown: Known = { agents: new Set(), skills: new Set() }

/** Token yang dikirim ke server untuk sebuah badge. */
export function tokenFor(kind: 'agent' | 'skill' | 'folder', value: string): string {
  if (kind === 'agent') return `@${value}`
  if (kind === 'skill') return `#${value}`
  return `[Folder: ${value}]`
}

/**
 * Memecah isi pesan menjadi teks dan badge inline: [Folder: /path], @agent, #skill.
 * @agent / #skill hanya menjadi badge bila slug-nya memang ada (known), supaya teks seperti "#123" tetap teks.
 */
export function segmentMessage(content: string, known: Known): Segment[] {
  const re = /\[Folder: ([^\]\n]+)\]|(^|[\s([{])([@#])([a-zA-Z0-9][a-zA-Z0-9._-]*)/g
  const out: Segment[] = []
  let last = 0
  const pushText = (t: string) => {
    if (t) out.push({ type: 'text', text: t })
  }
  let m: RegExpExecArray | null
  while ((m = re.exec(content))) {
    if (m[1] !== undefined) {
      pushText(content.slice(last, m.index))
      const path = m[1]
      out.push({ type: 'folder', path, name: path.replace(/\/+$/, '').split('/').pop() || path })
      last = m.index + m[0].length
      continue
    }
    const slug = m[4].replace(/[._-]+$/, '')
    const kind = m[3] === '@' ? 'agent' : 'skill'
    if (!(kind === 'agent' ? known.agents : known.skills).has(slug.toLowerCase())) continue
    const start = m.index + m[2].length
    pushText(content.slice(last, start))
    out.push({ type: kind, slug })
    last = start + 1 + slug.length
    re.lastIndex = last
  }
  pushText(content.slice(last))
  return out
}
