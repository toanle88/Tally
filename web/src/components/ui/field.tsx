import type { InputHTMLAttributes } from 'react'

import { classNames } from './class-names'

export interface FieldProps
  extends Omit<
    InputHTMLAttributes<HTMLInputElement>,
    'id' | 'size' | 'aria-describedby' | 'aria-invalid'
  > {
  id: string
  label: string
  description?: string
  error?: string
}

export function Field({
  id,
  label,
  description,
  error,
  className,
  ...rest
}: FieldProps) {
  const descriptionId = `${id}-description`
  const errorId = `${id}-error`
  const describedBy = [description ? descriptionId : null, error ? errorId : null]
    .filter(Boolean)
    .join(' ')

  return (
    <div className="form-control w-full gap-2">
      <label className="label cursor-pointer justify-start gap-2" htmlFor={id}>
        <span className="font-medium text-base-content">{label}</span>
        {rest.required ? <span aria-hidden="true" className="text-error">*</span> : null}
      </label>
      <input
        {...rest}
        id={id}
        className={classNames(
          'input input-bordered min-h-11 w-full border-2 border-base-300 bg-base-100 px-3 text-base-content shadow-sm placeholder:text-base-content/50 focus:border-primary focus:outline-2 focus:outline-offset-1 focus:outline-primary',
          error && 'input-error border-error focus:border-error focus:outline-error',
          className,
        )}
        aria-describedby={describedBy || undefined}
        aria-invalid={error ? true : undefined}
        aria-errormessage={error ? errorId : undefined}
      />
      {description ? (
        <p id={descriptionId} className="text-sm text-base-content/70">
          {description}
        </p>
      ) : null}
      {error ? (
        <p id={errorId} className="text-sm text-error">
          {error}
        </p>
      ) : null}
    </div>
  )
}
