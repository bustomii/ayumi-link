import { cva, type VariantProps } from 'class-variance-authority'
import type { ButtonHTMLAttributes } from 'react'
import { cn } from '@/lib/utils'

const buttonVariants = cva(
  'inline-flex items-center justify-center gap-2 rounded-md text-sm font-medium transition-colors disabled:pointer-events-none disabled:opacity-50',
  {
    variants: {
      variant: {
        default: 'bg-accent text-accent-foreground hover:bg-accent/90',
        ghost: 'hover:bg-muted text-foreground',
        outline: 'border border-border bg-transparent hover:bg-muted',
      },
      size: {
        default: 'h-9 px-4 py-2',
        sm: 'h-8 px-3 text-xs',
        icon: 'h-8 w-8',
      },
    },
    defaultVariants: {
      variant: 'default',
      size: 'default',
    },
  },
)

type Props = ButtonHTMLAttributes<HTMLButtonElement> &
  VariantProps<typeof buttonVariants>

export function Button({ className, variant, size, ...props }: Props) {
  return (
    <button className={cn(buttonVariants({ variant, size }), className)} {...props} />
  )
}
