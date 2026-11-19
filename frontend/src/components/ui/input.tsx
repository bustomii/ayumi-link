import type { InputHTMLAttributes, Ref } from 'react'
import { cn } from '@/lib/utils'

export function Input({
  className,
  ref,
  ...props
}: InputHTMLAttributes<HTMLInputElement> & { ref?: Ref<HTMLInputElement> }) {
  return (
    <input
      ref={ref}
      className={cn(
        'w-full rounded-lg border border-border bg-background px-3 py-2 text-sm outline-none',
        'placeholder:text-muted-foreground focus:border-primary disabled:opacity-60',
        className,
      )}
      {...props}
    />
  )
}
