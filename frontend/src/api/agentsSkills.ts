import { api } from './http'

export type Agent = {
  id: string
  slug: string
  name: string
  label?: string
  description?: string
  skillIds: string[]
  /** 'disk' = diimpor dari file yang dibuat lewat chat Claude. */
  source?: string
  createdAt: string
  updatedAt: string
}

export type Skill = {
  id: string
  slug: string
  name: string
  label?: string
  description?: string
  body: string
  source?: string
  createdAt: string
  updatedAt: string
}

export type AgentInput = {
  name: string
  slug?: string
  label?: string
  description?: string
  skillIds?: string[]
}

export type SkillInput = {
  name: string
  slug?: string
  label?: string
  description?: string
  body?: string
}

function asAgent(a: Agent): Agent {
  return { ...a, skillIds: Array.isArray(a.skillIds) ? a.skillIds : [] }
}

function asSkill(s: Skill): Skill {
  return { ...s, body: s.body ?? '' }
}

export const fetchAgents = () =>
  api<{ agents: Agent[] | null }>('/api/agents').then((r) =>
    (Array.isArray(r.agents) ? r.agents : []).map(asAgent),
  )

export const fetchAgent = (id: string) =>
  api<{ agent: Agent; skills: Skill[] | null }>(`/api/agents/${encodeURIComponent(id)}`).then((r) => ({
    agent: asAgent(r.agent),
    skills: (Array.isArray(r.skills) ? r.skills : []).map(asSkill),
  }))

export const createAgent = (body: AgentInput) =>
  api<Agent>('/api/agents', { method: 'POST', json: body }).then(asAgent)

export const updateAgent = (id: string, body: AgentInput) =>
  api<Agent>(`/api/agents/${encodeURIComponent(id)}`, { method: 'PUT', json: body }).then(asAgent)

export const deleteAgent = (id: string) =>
  api<{ ok: boolean }>(`/api/agents/${encodeURIComponent(id)}`, { method: 'DELETE' })

export const fetchSkills = () =>
  api<{ skills: Skill[] | null }>('/api/skills').then((r) =>
    (Array.isArray(r.skills) ? r.skills : []).map(asSkill),
  )

export const fetchSkill = (id: string) =>
  api<{ skill: Skill; agents: Agent[] | null }>(`/api/skills/${encodeURIComponent(id)}`).then((r) => ({
    skill: asSkill(r.skill),
    agents: (Array.isArray(r.agents) ? r.agents : []).map(asAgent),
  }))

export const createSkill = (body: SkillInput) =>
  api<Skill>('/api/skills', { method: 'POST', json: body }).then(asSkill)

export const updateSkill = (id: string, body: SkillInput) =>
  api<Skill>(`/api/skills/${encodeURIComponent(id)}`, { method: 'PUT', json: body }).then(asSkill)

export const deleteSkill = (id: string) =>
  api<{ ok: boolean }>(`/api/skills/${encodeURIComponent(id)}`, { method: 'DELETE' })

export const fetchGeneratePrompt = (kind: 'agent' | 'skill', name: string, description: string) =>
  api<{ kind: string; prompt: string }>('/api/agents-skills/generate-prompt', {
    method: 'POST',
    json: { kind, name, description },
  })

/** Ambil objek JSON dari blok ```json ... ``` dalam teks Claude. */
export function parseGenerateJSON(text: string): {
  agent?: AgentInput
  skills?: SkillInput[]
  skill?: SkillInput
} | null {
  const m = /```json\s*([\s\S]*?)```/i.exec(text) || /```\s*([\s\S]*?)```/.exec(text)
  const raw = (m ? m[1] : text).trim()
  try {
    return JSON.parse(raw) as { agent?: AgentInput; skills?: SkillInput[]; skill?: SkillInput }
  } catch {
    const start = raw.indexOf('{')
    const end = raw.lastIndexOf('}')
    if (start < 0 || end <= start) return null
    try {
      return JSON.parse(raw.slice(start, end + 1)) as {
        agent?: AgentInput
        skills?: SkillInput[]
        skill?: SkillInput
      }
    } catch {
      return null
    }
  }
}
