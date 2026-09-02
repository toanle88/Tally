import type { ButtonHTMLAttributes, ReactNode } from 'react'

import { classNames } from './class-names'
import type { ButtonVariant, ControlSize } from './types'

export interface ButtonProps
  extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant
  size?: ControlSize
  loading?: boolean
  children: ReactNode
}

const variantClasses: Record<ButtonVariant, string> = {
  primary: 'btn-primary',
  secondary: 'btn-secondary',
  neutral: 'btn-neutral',
  ghost: 'btn-ghost',
  danger: 'btn-error',
}

const sizeClasses: Record<ControlSize, string> = {
  sm: 'btn-sm',
  md: 'min-h-11',
  lg: 'btn-lg',
}

export function Button({
  variant = 'primary',
  size = 'md',
  loading = false,
  disabled = false,
  className,
  children,
  type = 'button',
  ...rest
}: ButtonProps) {
  return (
    <button
      {...rest}
      type={type}
      disabled={disabled || loading}
      aria-busy={loading || undefined}
      className={classNames(
        'btn max-w-full whitespace-normal focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary',
        variantClasses[variant],
        sizeClasses[size],
        className,
      )}
    >
      {loading ? (
        <span className="loading loading-spinner loading-sm" aria-hidden="true" />
      ) : null}
      <span>{children}</span>
    </button>
  )
}
