import {
  Brain,
  ClipboardList,
  Feather,
  Flame,
  Gem,
  Hand,
  Pencil,
  Signal,
  SignalHigh,
  SignalLow,
  SignalMedium,
  Sparkles,
  Zap,
  type LucideIcon,
} from 'lucide-react'

export type Option = { value: string; label: string; hint: string; icon: LucideIcon; tone: string }

export type Prefs = { permissionMode: string; model: string; effort: string }

export const MODELS: Option[] = [
  { value: 'claude-fable-5-2', label: 'Fable 5.2', hint: 'Paling mampu', icon: Gem, tone: 'text-fuchsia-400' },
  { value: 'claude-opus-5-5', label: 'Opus 5.5', hint: 'Penalaran mendalam', icon: Brain, tone: 'text-orange-400' },
  { value: 'claude-sonnet-5-5', label: 'Sonnet 5.5', hint: 'Seimbang untuk harian', icon: Sparkles, tone: 'text-sky-400' },
  { value: 'claude-haiku-5-5', label: 'Haiku 5.5', hint: 'Paling cepat dan hemat', icon: Feather, tone: 'text-emerald-400' },
]

/** Model lama, tampil di "Model lainnya". */
export const MORE_MODELS: Option[] = [
  { value: 'claude-fable-5-1', label: 'Fable 5.1', hint: 'Generasi sebelumnya', icon: Gem, tone: 'text-fuchsia-400' },
  { value: 'claude-fable-5', label: 'Fable 5', hint: 'Generasi sebelumnya', icon: Gem, tone: 'text-fuchsia-400' },
  { value: 'claude-opus-5', label: 'Opus 5', hint: 'Generasi sebelumnya', icon: Brain, tone: 'text-orange-400' },
  { value: 'claude-opus-4-8', label: 'Opus 4.8', hint: 'Generasi sebelumnya', icon: Brain, tone: 'text-orange-400' },
  { value: 'claude-opus-4-7', label: 'Opus 4.7', hint: 'Generasi sebelumnya', icon: Brain, tone: 'text-orange-400' },
  { value: 'claude-opus-4-6', label: 'Opus 4.6', hint: 'Generasi sebelumnya', icon: Brain, tone: 'text-orange-400' },
  { value: 'claude-sonnet-5', label: 'Sonnet 5', hint: 'Generasi sebelumnya', icon: Sparkles, tone: 'text-sky-400' },
  { value: 'claude-sonnet-4-6', label: 'Sonnet 4.6', hint: 'Generasi sebelumnya', icon: Sparkles, tone: 'text-sky-400' },
  { value: 'claude-haiku-4-5', label: 'Haiku 4.5', hint: 'Cepat, tanpa pengaturan effort', icon: Feather, tone: 'text-emerald-400' },
]

export const EFFORTS: Option[] = [
  { value: 'low', label: 'Low', hint: 'Cepat, sedikit berpikir', icon: SignalLow, tone: 'text-emerald-400' },
  { value: 'medium', label: 'Medium', hint: 'Seimbang', icon: SignalMedium, tone: 'text-sky-400' },
  { value: 'high', label: 'High', hint: 'Lebih teliti', icon: SignalHigh, tone: 'text-amber-400' },
  { value: 'xhigh', label: 'XHigh', hint: 'Sangat teliti', icon: Signal, tone: 'text-orange-400' },
  { value: 'max', label: 'Max', hint: 'Usaha maksimal, paling lambat', icon: Flame, tone: 'text-red-400' },
]

export const PERMISSION_MODES: Option[] = [
  { value: 'auto', label: 'Auto', hint: 'Tool diizinkan otomatis', icon: Zap, tone: 'text-amber-400' },
  { value: 'acceptEdits', label: 'Edit', hint: 'Terima edit file otomatis', icon: Pencil, tone: 'text-sky-400' },
  { value: 'manual', label: 'Manual', hint: 'Tool yang butuh izin ditolak', icon: Hand, tone: 'text-emerald-400' },
  { value: 'plan', label: 'Plan', hint: 'Hanya membuat rencana', icon: ClipboardList, tone: 'text-violet-400' },
]

export function findOption(options: Option[], value: string): Option {
  return options.find((o) => o.value === value) ?? options[0]
}
