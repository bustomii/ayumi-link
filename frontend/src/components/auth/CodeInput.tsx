import { useEffect, useRef } from 'react'
import { Input } from '@/components/ui/input'

type Props = {
  value: string
  onChange: (v: string) => void
  /** Dipanggil otomatis saat 6 digit terisi. */
  onComplete: (code: string) => void
  disabled?: boolean
}

/** Input kode TOTP 6 digit; kirim otomatis begitu lengkap. */
export function CodeInput({ value, onChange, onComplete, disabled }: Props) {
  const ref = useRef<HTMLInputElement>(null)

  useEffect(() => {
    ref.current?.focus()
  }, [])

  return (
    <Input
      ref={ref}
      value={value}
      disabled={disabled}
      inputMode="numeric"
      autoComplete="one-time-code"
      pattern="[0-9]*"
      maxLength={6}
      placeholder="000000"
      aria-label="Kode 6 digit dari aplikasi autentikator"
      className="py-3 text-center font-mono text-2xl tracking-[0.5em]"
      onChange={(e) => {
        const digits = e.target.value.replace(/\D/g, '').slice(0, 6)
        onChange(digits)
        if (digits.length === 6) onComplete(digits)
      }}
    />
  )
}
