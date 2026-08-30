import type { HTMLAttributes } from 'react'

import { classNames } from './class-names'
import type { SemanticState } from './types'

export interface StatusBadgeProps extends HTMLAttributes<HTMLSpanElement> {
  state: SemanticState
  label?: string
  announce?: boolean
}

const stateLabels: Record<SemanticState, string> = {
  success: 'Success',
  warning: 'Warning',
  error: 'Error',
  info: 'Info',
  pending: 'Pending',
  reconciled: 'Reconciled',
  restricted: 'Restricted',
  disabled: 'Disabled',
}

const stateClasses: Record<SemanticState, string> = {
  success: 'bg-[var(--color-state-success)] text-[var(--color-state-success-content)]',
  warning: 'bg-[var(--color-state-warning)] text-[var(--color-state-warning-content)]',
  error: 'bg-[var(--color-state-error)] text-[var(--color-state-error-content)]',
  info: 'bg-[var(--color-state-info)] text-[var(--color-state-info-content)]',
  pending: 'bg-[var(--color-state-pending)] text-[var(--color-state-pending-content)]',
  reconciled: 'bg-[var(--color-state-reconciled)] text-[var(--color-state-reconciled-content)]',
  restricted: 'bg-[var(--color-state-restricted)] text-[var(--color-state-restricted-content)]',
  disabled: 'bg-[var(--color-state-disabled)] text-[var(--color-state-disabled-content)]',
}

export function StatusBadge({
  state,
  label = stateLabels[state],
  announce = false,
  className,
  ...rest
}: StatusBadgeProps) {
  return (
    <span
      {...rest}
      role={announce ? 'status' : undefined}
      aria-label={label}
      data-state={state}
      className={classNames(
        'badge min-h-6 min-w-16 justify-center whitespace-nowrap border-0 px-2 py-1 text-xs font-semibold leading-5',
        stateClasses[state],
        className,
      )}
    >
      {label}
    </span>
  )
}
