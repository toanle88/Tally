import type { SelectHTMLAttributes } from 'react'

import { classNames } from './class-names'

export type SelectProps = SelectHTMLAttributes<HTMLSelectElement>

export function Select({ className, children, ...rest }: SelectProps) {
  return (
    <div className="relative">
      <select
        {...rest}
        className={classNames(
          'select select-bordered min-h-11 w-full max-w-full truncate appearance-none border-2 border-base-300 bg-base-100 bg-none pr-12 text-base-content focus:border-primary focus:outline-2 focus:outline-offset-1 focus:outline-primary',
          className,
        )}
      >
        {children}
      </select>
      <svg
        aria-hidden="true"
        className="pointer-events-none absolute right-3 top-1/2 h-4 w-4 -translate-y-1/2 text-base-content/70"
        viewBox="0 0 20 20"
        fill="none"
        stroke="currentColor"
        strokeWidth="2"
      >
        <path d="m5 7 5 5 5-5" strokeLinecap="round" strokeLinejoin="round" />
      </svg>
    </div>
  )
}
