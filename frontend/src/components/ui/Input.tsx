import { cn } from '@/lib/utils'
import { forwardRef, type InputHTMLAttributes } from 'react'

interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  label?: string
  error?: string
  hint?: string
}

const Input = forwardRef<HTMLInputElement, InputProps>(
  ({ label, error, hint, className, id, ...props }, ref) => {
    const inputId = id ?? label?.toLowerCase().replace(/\s+/g, '-')

    return (
      <div className="flex flex-col gap-1">
        {label && (
          <label htmlFor={inputId} className="text-sm font-medium text-slate-300">
            {label}
          </label>
        )}
        <input
          ref={ref}
          id={inputId}
          {...props}
          className={cn(
            'w-full rounded-lg bg-navy-950 border px-3.5 py-2.5 text-sm text-white placeholder:text-slate-500',
            'transition-colors focus:outline-none focus:ring-2 focus:ring-brand-cyan focus:border-transparent',
            error ? 'border-danger' : 'border-navy-600 hover:border-navy-600/80',
            className,
          )}
        />
        {error && <p className="text-xs text-danger">{error}</p>}
        {hint && !error && <p className="text-xs text-muted">{hint}</p>}
      </div>
    )
  },
)

Input.displayName = 'Input'
export default Input
