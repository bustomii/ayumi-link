import { useEffect, useState } from 'react'
import { fetchAgents, fetchSkills, type Agent, type Skill } from '@/api/agentsSkills'
import { emptyKnown, type Known } from '@/lib/chat-tags'

type Index = { agents: Agent[]; skills: Skill[]; known: Known }

const empty: Index = { agents: [], skills: [], known: emptyKnown }
let cache: { at: number; p: Promise<Index> } | null = null

function load(): Promise<Index> {
  if (cache && Date.now() - cache.at < 30_000) return cache.p
  const p = Promise.all([fetchAgents(), fetchSkills()])
    .then(([a, s]) => {
      const agents = Array.isArray(a) ? a : []
      const skills = Array.isArray(s) ? s : []
      return {
        agents,
        skills,
        known: {
          agents: new Set(agents.map((x) => x.slug.toLowerCase())),
          skills: new Set(skills.map((x) => x.slug.toLowerCase())),
        },
      }
    })
    .catch(() => {
      cache = null
      return empty
    })
  cache = { at: Date.now(), p }
  return p
}

/** Daftar agent & skill (di-cache 30 detik) untuk autocomplete dan badge di chat. */
export function useMentionIndex(): Index {
  const [idx, setIdx] = useState<Index>(empty)
  useEffect(() => {
    let off = false
    void load().then((v) => {
      if (!off) setIdx(v)
    })
    return () => {
      off = true
    }
  }, [])
  return idx
}
