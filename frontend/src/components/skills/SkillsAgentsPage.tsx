import { ArrowLeft, Bot, Menu, Pencil, Plus, Search, Sparkles, Trash2, Wrench, X } from 'lucide-react'
import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react'
import { AccountMenu } from '@/components/AccountMenu'
import {
  createAgent,
  createSkill,
  deleteAgent,
  deleteSkill,
  fetchAgents,
  fetchGeneratePrompt,
  fetchSkills,
  parseGenerateJSON,
  updateAgent,
  updateSkill,
  type Agent,
  type AgentInput,
  type Skill,
  type SkillInput,
} from '@/api/agentsSkills'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Skeleton } from '@/components/ui/skeleton'
import { useErrorDialog } from '@/hooks/useErrorDialog'
import { cn } from '@/lib/utils'

type Tab = 'agents' | 'skills'
type Props = {
  onBack: () => void
  onOpenSidebar: () => void
  /** Buka chat baru dengan prompt generate (Workspace kirim pesan). */
  onGenerateChat: (prompt: string, titleHint: string) => void
  username: string
  onOpenProfile: () => void
  onLogout: () => void
}

export function SkillsAgentsPage({ onBack, onOpenSidebar, onGenerateChat, username, onOpenProfile, onLogout }: Props) {
  const errors = useErrorDialog()
  const showError = errors.show
  const [tab, setTab] = useState<Tab>('agents')
  const [agents, setAgents] = useState<Agent[]>([])
  const [skills, setSkills] = useState<Skill[]>([])
  const [loading, setLoading] = useState(true)
  const [selectedAgent, setSelectedAgent] = useState<string | null>(null)
  const [selectedSkill, setSelectedSkill] = useState<string | null>(null)
  const [editAgent, setEditAgent] = useState<Agent | null>(null)
  const [editSkill, setEditSkill] = useState<Skill | null>(null)
  const [creating, setCreating] = useState<'agent' | 'skill' | null>(null)
  const [genOpen, setGenOpen] = useState<'agent' | 'skill' | null>(null)
  const [importOpen, setImportOpen] = useState(false)
  const [busy, setBusy] = useState(false)
  const [query, setQuery] = useState('')

  const reload = useCallback(async () => {
    setLoading(true)
    try {
      const [a, s] = await Promise.all([fetchAgents(), fetchSkills()])
      setAgents(Array.isArray(a) ? a : [])
      setSkills(Array.isArray(s) ? s : [])
    } catch (e) {
      showError(e, 'Skills & Agents', () => void reload())
    } finally {
      setLoading(false)
    }
  }, [showError])

  useEffect(() => {
    void reload()
  }, [reload])

  const agentList = Array.isArray(agents) ? agents : []
  const skillList = Array.isArray(skills) ? skills : []
  const q = query.trim().toLowerCase()

  const filteredAgents = useMemo(() => {
    if (!q) return agentList
    return agentList.filter((a) => {
      const hay = [a.name, a.slug, a.label, a.description, `@${a.slug}`].filter(Boolean).join(' ').toLowerCase()
      return hay.includes(q)
    })
  }, [agentList, q])

  const filteredSkills = useMemo(() => {
    if (!q) return skillList
    return skillList.filter((s) => {
      const hay = [s.name, s.slug, s.label, s.description, s.body, `#${s.slug}`].filter(Boolean).join(' ').toLowerCase()
      return hay.includes(q)
    })
  }, [skillList, q])

  const skillById = useMemo(() => {
    const m = new Map<string, Skill>()
    for (const s of skillList) m.set(s.id, s)
    return m
  }, [skillList])

  const agent = agentList.find((a) => a.id === selectedAgent) ?? null
  const skill = skillList.find((s) => s.id === selectedSkill) ?? null
  const agentSkillIds = agent && Array.isArray(agent.skillIds) ? agent.skillIds : []
  const detailOpen = (tab === 'agents' && !!agent) || (tab === 'skills' && !!skill)
  const detailTitle = tab === 'agents' ? (agent?.name ?? 'Agent') : (skill?.name ?? 'Skill')

  const closeDetail = () => {
    setSelectedAgent(null)
    setSelectedSkill(null)
  }

  const switchTab = (next: Tab) => {
    setTab(next)
    closeDetail()
  }

  return (
    <div className="relative flex min-h-0 flex-1 flex-col overflow-hidden bg-[radial-gradient(ellipse_at_top,_#151821_0%,_#0c0d10_50%)]">
      <header
        className={cn(
          'flex shrink-0 items-center gap-2 border-b border-border/70 px-3 py-2.5',
          detailOpen && 'hidden md:flex',
        )}
      >
        <Button type="button" variant="ghost" size="icon" className="h-9 w-9 md:hidden" onClick={onOpenSidebar} aria-label="Menu">
          <Menu className="h-5 w-5" />
        </Button>
        <Button type="button" variant="ghost" size="icon" className="h-9 w-9" onClick={onBack} aria-label="Kembali">
          <ArrowLeft className="h-5 w-5" />
        </Button>
        <div className="min-w-0 flex-1">
          <h1 className="truncate text-sm font-semibold">Skills & Agents</h1>
          <p className="truncate text-[11px] text-muted-foreground">Tim agent + skill · @agent #skill</p>
        </div>
        <Button type="button" variant="outline" size="sm" className="hidden shrink-0 sm:inline-flex" onClick={() => setImportOpen(true)}>
          Impor JSON
        </Button>
        <Button
          type="button"
          size="icon"
          className="h-9 w-9 shrink-0"
          title="Buat baru"
          onClick={() => setCreating(tab === 'agents' ? 'agent' : 'skill')}
        >
          <Plus className="h-4 w-4" />
        </Button>
        <AccountMenu
          username={username}
          onOpenProfile={onOpenProfile}
          onOpenSkills={() => {}}
          onLogout={onLogout}
          skillsActive
        />
      </header>

      <div className={cn('flex shrink-0 gap-1 border-b border-border/50 px-3 py-2', detailOpen && 'hidden md:flex')}>
        <TabBtn active={tab === 'agents'} onClick={() => switchTab('agents')} icon={<Bot className="h-3.5 w-3.5" />} label="Agents" />
        <TabBtn active={tab === 'skills'} onClick={() => switchTab('skills')} icon={<Wrench className="h-3.5 w-3.5" />} label="Skills" />
        <div className="flex-1" />
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="gap-1.5 text-xs"
          onClick={() => setGenOpen(tab === 'agents' ? 'agent' : 'skill')}
        >
          <Sparkles className="h-3.5 w-3.5" />
          <span className="hidden sm:inline">Generate dengan Claude</span>
          <span className="sm:hidden">Generate</span>
        </Button>
      </div>

      <div className="relative flex min-h-0 flex-1 overflow-hidden">
        {/* Daftar: full di mobile; disembunyikan saat detail terbuka di mobile */}
        <div
          className={cn(
            'flex min-h-0 w-full flex-col md:max-w-sm md:shrink-0 md:border-r md:border-border/50',
            detailOpen && 'hidden md:flex',
          )}
        >
          <div className="shrink-0 border-b border-border/40 p-2">
            <label className="relative block">
              <Search className="pointer-events-none absolute top-1/2 left-2.5 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
              <input
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder={tab === 'agents' ? 'Cari agent…' : 'Cari skill…'}
                className="w-full rounded-lg border border-border bg-background/60 py-2 pr-8 pl-8 text-sm outline-none placeholder:text-muted-foreground focus:border-primary"
                aria-label="Cari"
              />
              {query && (
                <button
                  type="button"
                  className="absolute top-1/2 right-1.5 -translate-y-1/2 rounded p-1 text-muted-foreground hover:bg-white/5 hover:text-foreground"
                  onClick={() => setQuery('')}
                  aria-label="Hapus pencarian"
                >
                  <X className="h-3.5 w-3.5" />
                </button>
              )}
            </label>
            {!loading && q && (
              <p className="mt-1.5 px-0.5 text-[11px] text-muted-foreground">
                {tab === 'agents'
                  ? `${filteredAgents.length} dari ${agentList.length} agent`
                  : `${filteredSkills.length} dari ${skillList.length} skill`}
              </p>
            )}
          </div>
          <ScrollArea className="min-h-0 flex-1">
            <div className="space-y-1 p-2 pb-[max(0.5rem,env(safe-area-inset-bottom))]">
              {loading ? (
                Array.from({ length: 5 }).map((_, i) => <Skeleton key={i} className="h-14 w-full rounded-xl" />)
              ) : tab === 'agents' ? (
                agentList.length === 0 ? (
                  <Empty hint="Belum ada agent. Tekan + atau Generate." />
                ) : filteredAgents.length === 0 ? (
                  <Empty hint={`Tidak ada agent untuk “${query.trim()}”.`} />
                ) : (
                  filteredAgents.map((a) => (
                    <button
                      key={a.id}
                      type="button"
                      onClick={() => {
                        setSelectedAgent(a.id)
                        setSelectedSkill(null)
                      }}
                      className={cn(
                        'flex w-full flex-col gap-0.5 rounded-xl px-3 py-2.5 text-left transition-colors hover:bg-white/5',
                        selectedAgent === a.id && 'bg-accent/15',
                      )}
                    >
                      <span className="flex items-center gap-2">
                        <span className="truncate text-sm font-medium">{a.name}</span>
                        {a.label && <LabelChip text={a.label} />}
                        {a.source === 'disk' && <LabelChip text="dari chat" />}
                      </span>
                      <span className="truncate text-[11px] text-muted-foreground">
                        @{a.slug} · {(a.skillIds ?? []).length} skill
                      </span>
                    </button>
                  ))
                )
              ) : skillList.length === 0 ? (
                <Empty hint="Belum ada skill. Tekan + atau Generate." />
              ) : filteredSkills.length === 0 ? (
                <Empty hint={`Tidak ada skill untuk “${query.trim()}”.`} />
              ) : (
                filteredSkills.map((s) => (
                  <button
                    key={s.id}
                    type="button"
                    onClick={() => {
                      setSelectedSkill(s.id)
                      setSelectedAgent(null)
                    }}
                    className={cn(
                      'flex w-full flex-col gap-0.5 rounded-xl px-3 py-2.5 text-left transition-colors hover:bg-white/5',
                      selectedSkill === s.id && 'bg-accent/15',
                    )}
                  >
                    <span className="flex items-center gap-2">
                      <span className="truncate text-sm font-medium">{s.name}</span>
                      {s.label && <LabelChip text={s.label} />}
                      {s.source === 'disk' && <LabelChip text="dari chat" />}
                    </span>
                    <span className="truncate text-[11px] text-muted-foreground">#{s.slug}</span>
                  </button>
                ))
              )}
            </div>
          </ScrollArea>
        </div>

        {/* Detail: full layar di mobile saat terbuka; panel kanan di desktop */}
        <div
          className={cn(
            'min-h-0 min-w-0 flex-col bg-background',
            detailOpen
              ? 'absolute inset-0 z-20 flex md:static md:z-auto md:flex md:flex-1'
              : 'hidden md:flex md:flex-1',
          )}
        >
          {detailOpen && (
            <div className="flex shrink-0 items-center gap-2 border-b border-border/70 px-3 py-2.5 md:hidden">
              <Button type="button" variant="ghost" size="icon" className="h-9 w-9" onClick={closeDetail} aria-label="Tutup">
                <ArrowLeft className="h-5 w-5" />
              </Button>
              <div className="min-w-0 flex-1">
                <h2 className="truncate text-sm font-semibold">{detailTitle}</h2>
                <p className="truncate text-[11px] text-muted-foreground">
                  {tab === 'agents' ? `@${agent?.slug ?? ''}` : `#${skill?.slug ?? ''}`}
                </p>
              </div>
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="h-9 w-9"
                onClick={() => (tab === 'agents' && agent ? setEditAgent(agent) : skill && setEditSkill(skill))}
                aria-label="Edit"
              >
                <Pencil className="h-4 w-4" />
              </Button>
              <Button type="button" variant="ghost" size="icon" className="h-9 w-9" onClick={closeDetail} aria-label="Tutup detail">
                <X className="h-5 w-5" />
              </Button>
            </div>
          )}
          <ScrollArea className="min-h-0 flex-1">
            <div className="mx-auto max-w-2xl p-4 pb-[max(1rem,env(safe-area-inset-bottom))]">
              {tab === 'agents' && agent ? (
                <AgentDetail
                  agent={agent}
                  skills={agentSkillIds.map((id) => skillById.get(id)).filter(Boolean) as Skill[]}
                  onEdit={() => setEditAgent(agent)}
                  onDelete={async () => {
                    if (!confirm(`Hapus agent ${agent.name}?`)) return
                    setBusy(true)
                    try {
                      await deleteAgent(agent.id)
                      closeDetail()
                      await reload()
                    } catch (e) {
                      showError(e instanceof Error ? e.message : 'Gagal hapus', 'Hapus agent')
                    } finally {
                      setBusy(false)
                    }
                  }}
                  busy={busy}
                />
              ) : tab === 'skills' && skill ? (
                <SkillDetail
                  skill={skill}
                  agents={agentList.filter((a) => (a.skillIds ?? []).includes(skill.id))}
                  onEdit={() => setEditSkill(skill)}
                  onDelete={async () => {
                    if (!confirm(`Hapus skill ${skill.name}?`)) return
                    setBusy(true)
                    try {
                      await deleteSkill(skill.id)
                      closeDetail()
                      await reload()
                    } catch (e) {
                      showError(e instanceof Error ? e.message : 'Gagal hapus', 'Hapus skill')
                    } finally {
                      setBusy(false)
                    }
                  }}
                  busy={busy}
                />
              ) : (
                <div className="flex flex-col items-center justify-center gap-2 py-16 text-center text-sm text-muted-foreground">
                  <Bot className="h-8 w-8 opacity-40" />
                  Pilih item di kiri, atau buat baru dengan +
                </div>
              )}
            </div>
          </ScrollArea>
        </div>
      </div>

      {creating && (
        <EditorModal
          kind={creating}
          title={creating === 'agent' ? 'Agent baru' : 'Skill baru'}
          skills={skillList}
          onClose={() => setCreating(null)}
          onSave={async (data) => {
            setBusy(true)
            try {
              if (creating === 'agent') {
                const a = await createAgent(data as AgentInput)
                setSelectedAgent(a.id)
                setTab('agents')
              } else {
                const s = await createSkill(data as SkillInput)
                setSelectedSkill(s.id)
                setTab('skills')
              }
              setCreating(null)
              await reload()
            } catch (e) {
              showError(e instanceof Error ? e.message : 'Gagal menyimpan', 'Simpan')
            } finally {
              setBusy(false)
            }
          }}
          busy={busy}
        />
      )}

      {editAgent && (
        <EditorModal
          kind="agent"
          title="Edit agent"
          initial={editAgent}
          skills={skillList}
          onClose={() => setEditAgent(null)}
          onSave={async (data) => {
            setBusy(true)
            try {
              await updateAgent(editAgent.id, data as AgentInput)
              setEditAgent(null)
              await reload()
            } catch (e) {
              showError(e instanceof Error ? e.message : 'Gagal menyimpan', 'Edit agent')
            } finally {
              setBusy(false)
            }
          }}
          busy={busy}
        />
      )}

      {editSkill && (
        <EditorModal
          kind="skill"
          title="Edit skill"
          initial={editSkill}
          skills={skillList}
          onClose={() => setEditSkill(null)}
          onSave={async (data) => {
            setBusy(true)
            try {
              await updateSkill(editSkill.id, data as SkillInput)
              setEditSkill(null)
              await reload()
            } catch (e) {
              showError(e instanceof Error ? e.message : 'Gagal menyimpan', 'Edit skill')
            } finally {
              setBusy(false)
            }
          }}
          busy={busy}
        />
      )}

      {genOpen && (
        <GenerateModal
          kind={genOpen}
          onClose={() => setGenOpen(null)}
          onGo={async (name, description) => {
            const { prompt } = await fetchGeneratePrompt(genOpen, name, description)
            setGenOpen(null)
            onGenerateChat(prompt, name || (genOpen === 'agent' ? 'Agent baru' : 'Skill baru'))
          }}
        />
      )}

      {importOpen && (
        <ImportModal
          onClose={() => setImportOpen(false)}
          onImport={async (text) => {
            const parsed = parseGenerateJSON(text)
            if (!parsed) throw new Error('JSON tidak valid')
            setBusy(true)
            try {
              if (parsed.skill) {
                const s = await createSkill(parsed.skill)
                setTab('skills')
                setSelectedSkill(s.id)
              }
              const createdSkillIds: string[] = []
              if (parsed.skills?.length) {
                for (const sk of parsed.skills) {
                  const s = await createSkill(sk)
                  createdSkillIds.push(s.id)
                }
              }
              if (parsed.agent) {
                const a = await createAgent({
                  ...parsed.agent,
                  skillIds: [...(parsed.agent.skillIds ?? []), ...createdSkillIds],
                })
                setTab('agents')
                setSelectedAgent(a.id)
              }
              if (!parsed.agent && !parsed.skill && !parsed.skills?.length) {
                throw new Error('JSON tidak berisi agent/skill')
              }
              setImportOpen(false)
              await reload()
            } finally {
              setBusy(false)
            }
          }}
        />
      )}
      {errors.dialog}
    </div>
  )
}

function TabBtn({ active, onClick, icon, label }: { active: boolean; onClick: () => void; icon: ReactNode; label: string }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        'inline-flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-medium transition-colors',
        active ? 'bg-accent/20 text-foreground' : 'text-muted-foreground hover:bg-white/5',
      )}
    >
      {icon}
      {label}
    </button>
  )
}

function LabelChip({ text }: { text: string }) {
  return (
    <span className="shrink-0 rounded bg-white/10 px-1.5 py-0.5 text-[10px] font-medium text-muted-foreground">{text}</span>
  )
}

function Empty({ hint }: { hint: string }) {
  return <p className="px-3 py-8 text-center text-xs text-muted-foreground">{hint}</p>
}

function AgentDetail({
  agent,
  skills,
  onEdit,
  onDelete,
  busy,
}: {
  agent: Agent
  skills: Skill[]
  onEdit: () => void
  onDelete: () => void
  busy: boolean
}) {
  return (
    <div className="space-y-4">
      <div className="flex items-start gap-3">
        <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-primary/15 text-primary">
          <Bot className="h-5 w-5" />
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2">
            <h2 className="text-lg font-semibold">{agent.name}</h2>
            {agent.label && <LabelChip text={agent.label} />}
          </div>
          <p className="text-xs text-muted-foreground">@{agent.slug}</p>
        </div>
        <Button type="button" variant="ghost" size="icon" className="h-8 w-8" onClick={onEdit} aria-label="Edit">
          <Pencil className="h-4 w-4" />
        </Button>
        <Button type="button" variant="ghost" size="icon" className="h-8 w-8 text-red-400" disabled={busy} onClick={onDelete} aria-label="Hapus">
          <Trash2 className="h-4 w-4" />
        </Button>
      </div>
      {agent.description && <p className="text-sm text-muted-foreground whitespace-pre-wrap">{agent.description}</p>}
      <div>
        <h3 className="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">Skills</h3>
        {skills.length === 0 ? (
          <p className="text-xs text-muted-foreground">Belum ada skill terpasang. Edit agent untuk menambahkan.</p>
        ) : (
          <ul className="space-y-2">
            {skills.map((s) => (
              <li key={s.id} className="rounded-xl border border-border/60 bg-card/40 px-3 py-2">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-medium">{s.name}</span>
                  {s.label && <LabelChip text={s.label} />}
                  <span className="text-[11px] text-muted-foreground">#{s.slug}</span>
                </div>
                {s.description && <p className="mt-1 text-xs text-muted-foreground line-clamp-2">{s.description}</p>}
              </li>
            ))}
          </ul>
        )}
      </div>
      <p className="text-[11px] text-muted-foreground">Di chat: ketik <code className="rounded bg-white/5 px-1">@{agent.slug}</code></p>
    </div>
  )
}

function SkillDetail({
  skill,
  agents,
  onEdit,
  onDelete,
  busy,
}: {
  skill: Skill
  agents: Agent[]
  onEdit: () => void
  onDelete: () => void
  busy: boolean
}) {
  return (
    <div className="space-y-4">
      <div className="flex items-start gap-3">
        <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-amber-500/15 text-amber-400">
          <Wrench className="h-5 w-5" />
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2">
            <h2 className="text-lg font-semibold">{skill.name}</h2>
            {skill.label && <LabelChip text={skill.label} />}
          </div>
          <p className="text-xs text-muted-foreground">#{skill.slug}</p>
        </div>
        <Button type="button" variant="ghost" size="icon" className="h-8 w-8" onClick={onEdit} aria-label="Edit">
          <Pencil className="h-4 w-4" />
        </Button>
        <Button type="button" variant="ghost" size="icon" className="h-8 w-8 text-red-400" disabled={busy} onClick={onDelete} aria-label="Hapus">
          <Trash2 className="h-4 w-4" />
        </Button>
      </div>
      {skill.description && <p className="text-sm text-muted-foreground whitespace-pre-wrap">{skill.description}</p>}
      {agents.length > 0 && (
        <div className="flex flex-wrap gap-1.5">
          {agents.map((a) => (
            <span key={a.id} className="rounded-full border border-border/60 px-2 py-0.5 text-[11px]">
              @{a.slug}
            </span>
          ))}
        </div>
      )}
      <div>
        <h3 className="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">Detail skill</h3>
        <pre className="max-h-[50vh] overflow-auto rounded-xl border border-border/60 bg-[#0a0b0e] p-3 text-xs leading-relaxed whitespace-pre-wrap">
          {skill.body || '(kosong)'}
        </pre>
      </div>
      <p className="text-[11px] text-muted-foreground">Di chat: ketik <code className="rounded bg-white/5 px-1">#{skill.slug}</code></p>
    </div>
  )
}

function EditorModal({
  kind,
  title,
  initial,
  skills,
  onClose,
  onSave,
  busy,
}: {
  kind: 'agent' | 'skill'
  title: string
  initial?: Agent | Skill
  skills: Skill[]
  onClose: () => void
  onSave: (data: AgentInput | SkillInput) => Promise<void>
  busy: boolean
}) {
  const [name, setName] = useState(initial?.name ?? '')
  const [slug, setSlug] = useState(initial?.slug ?? '')
  const [label, setLabel] = useState(initial?.label ?? '')
  const [description, setDescription] = useState(initial?.description ?? '')
  const [body, setBody] = useState(kind === 'skill' ? ((initial as Skill | undefined)?.body ?? '') : '')
  const [skillIds, setSkillIds] = useState<string[]>(
    kind === 'agent' ? (Array.isArray((initial as Agent | undefined)?.skillIds) ? (initial as Agent).skillIds : []) : [],
  )

  return (
    <Modal title={title} onClose={onClose}>
      <Field label="Nama" value={name} onChange={setName} required />
      <Field label="Slug" value={slug} onChange={setSlug} placeholder="otomatis dari nama" />
      <Field label="Label" value={label} onChange={setLabel} placeholder="opsional, singkat" />
      <Field label="Deskripsi" value={description} onChange={setDescription} multiline />
      {kind === 'skill' && <Field label="Body (instruksi detail)" value={body} onChange={setBody} multiline tall />}
      {kind === 'agent' && (
        <div>
          <p className="mb-1.5 text-xs font-medium text-muted-foreground">Skills terpasang</p>
          <div className="max-h-40 space-y-1 overflow-y-auto rounded-lg border border-border/60 p-2">
            {skills.length === 0 ? (
              <p className="text-xs text-muted-foreground">Belum ada skill. Buat skill dulu.</p>
            ) : (
              skills.map((s) => (
                <label key={s.id} className="flex cursor-pointer items-center gap-2 rounded-md px-2 py-1.5 text-sm hover:bg-white/5">
                  <input
                    type="checkbox"
                    checked={skillIds.includes(s.id)}
                    onChange={(e) =>
                      setSkillIds((ids) => (e.target.checked ? [...ids, s.id] : ids.filter((x) => x !== s.id)))
                    }
                  />
                  <span className="min-w-0 flex-1 truncate">{s.name}</span>
                  <span className="text-[11px] text-muted-foreground">#{s.slug}</span>
                </label>
              ))
            )}
          </div>
        </div>
      )}
      <div className="flex justify-end gap-2 pt-2">
        <Button type="button" variant="ghost" onClick={onClose}>
          Batal
        </Button>
        <Button
          type="button"
          disabled={busy || !name.trim()}
          onClick={() =>
            void onSave(
              kind === 'agent'
                ? { name, slug: slug || undefined, label, description, skillIds }
                : { name, slug: slug || undefined, label, description, body },
            )
          }
        >
          Simpan
        </Button>
      </div>
    </Modal>
  )
}

function GenerateModal({
  kind,
  onClose,
  onGo,
}: {
  kind: 'agent' | 'skill'
  onClose: () => void
  onGo: (name: string, description: string) => Promise<void>
}) {
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState<string | null>(null)
  return (
    <Modal title={kind === 'agent' ? 'Generate agent dengan Claude' : 'Generate skill dengan Claude'} onClose={onClose}>
      <p className="text-xs text-muted-foreground">
        Ayumi membuka chat baru dengan prompt generate. Setelah Claude mengeluarkan blok JSON, salin ke <strong>Impor JSON</strong> atau
        buat manual dari usulannya.
      </p>
      <Field label="Nama (opsional)" value={name} onChange={setName} placeholder={kind === 'agent' ? 'mis. DevOps' : 'mis. Deploy K8s'} />
      <Field
        label="Deskripsi"
        value={description}
        onChange={setDescription}
        multiline
        placeholder="Jelaskan peran / apa yang harus bisa dilakukan…"
        required
      />
      {err && <p className="text-xs text-red-400">{err}</p>}
      <div className="flex justify-end gap-2 pt-2">
        <Button type="button" variant="ghost" onClick={onClose}>
          Batal
        </Button>
        <Button
          type="button"
          disabled={busy || !description.trim()}
          className="gap-1.5"
          onClick={() => {
            setBusy(true)
            setErr(null)
            void onGo(name, description)
              .catch((e) => setErr(e instanceof Error ? e.message : 'Gagal'))
              .finally(() => setBusy(false))
          }}
        >
          <Sparkles className="h-3.5 w-3.5" />
          Buka chat Claude
        </Button>
      </div>
    </Modal>
  )
}

function ImportModal({ onClose, onImport }: { onClose: () => void; onImport: (text: string) => Promise<void> }) {
  const [text, setText] = useState('')
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState<string | null>(null)
  return (
    <Modal title="Impor dari JSON Claude" onClose={onClose}>
      <Field label="Tempel blok JSON" value={text} onChange={setText} multiline tall placeholder='{"agent":{...},"skills":[...]}' />
      {err && <p className="text-xs text-red-400">{err}</p>}
      <div className="flex justify-end gap-2 pt-2">
        <Button type="button" variant="ghost" onClick={onClose}>
          Batal
        </Button>
        <Button
          type="button"
          disabled={busy || !text.trim()}
          onClick={() => {
            setBusy(true)
            setErr(null)
            void onImport(text)
              .catch((e) => setErr(e instanceof Error ? e.message : 'Gagal impor'))
              .finally(() => setBusy(false))
          }}
        >
          Impor
        </Button>
      </div>
    </Modal>
  )
}

function Modal({ title, onClose, children }: { title: string; onClose: () => void; children: ReactNode }) {
  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center bg-black/60 p-3 sm:items-center" role="dialog">
      <div className="max-h-[90vh] w-full max-w-lg overflow-y-auto rounded-2xl border border-border bg-card p-4 shadow-xl">
        <div className="mb-3 flex items-center justify-between gap-2">
          <h2 className="text-sm font-semibold">{title}</h2>
          <button type="button" className="rounded-lg p-1 hover:bg-white/5" onClick={onClose} aria-label="Tutup">
            <X className="h-4 w-4" />
          </button>
        </div>
        <div className="space-y-3">{children}</div>
      </div>
    </div>
  )
}

function Field({
  label,
  value,
  onChange,
  placeholder,
  multiline,
  tall,
  required,
}: {
  label: string
  value: string
  onChange: (v: string) => void
  placeholder?: string
  multiline?: boolean
  tall?: boolean
  required?: boolean
}) {
  const cls =
    'w-full rounded-lg border border-border bg-background/60 px-3 py-2 text-sm outline-none focus:border-primary'
  return (
    <label className="block space-y-1">
      <span className="text-xs font-medium text-muted-foreground">
        {label}
        {required ? ' *' : ''}
      </span>
      {multiline ? (
        <textarea
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder={placeholder}
          rows={tall ? 10 : 3}
          className={cn(cls, 'resize-y')}
        />
      ) : (
        <input value={value} onChange={(e) => onChange(e.target.value)} placeholder={placeholder} className={cls} />
      )}
    </label>
  )
}
