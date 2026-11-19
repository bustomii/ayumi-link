import { AlertCircle, Eye, EyeOff } from 'lucide-react'
import { useState, type InputHTMLAttributes, type ReactNode } from 'react'
import { Input } from '@/components/ui/input'

type FieldProps = { label: string; hint?: string; children: ReactNode }

export function Field({ label, hint, children }: FieldProps) {
  return (
    <label className="block space-y-1.5">
      <span className="text-xs font-medium">{label}</span>
      {children}
      {hint && <span className="block text-[11px] text-muted-foreground">{hint}</span>}
    </label>
  )
}

/** Input password dengan tombol tampilkan/sembunyikan. */
export function PasswordInput(props: InputHTMLAttributes<HTMLInputElement>) {
  const [show, setShow] = useState(false)
  return (
    <div className="relative">
      <Input {...props} type={show ? 'text' : 'password'} className="pr-10" />
      <button
        type="button"
        onClick={() => setShow((s) => !s)}
        aria-label={show ? 'Sembunyikan password' : 'Tampilkan password'}
        className="absolute inset-y-0 right-0 flex w-10 items-center justify-center text-muted-foreground hover:text-foreground"
        tabIndex={-1}
      >
        {show ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
      </button>
    </div>
  )
}

export function FormError({ children }: { children: ReactNode }) {
  return (
    <p
      role="alert"
      className="flex items-start gap-2 rounded-lg border border-red-900/50 bg-red-950/40 px-3 py-2 text-xs text-red-300"
    >
      <AlertCircle className="mt-px h-4 w-4 shrink-0" />
      <span className="first-letter:uppercase">{children}</span>
    </p>
  )
}
