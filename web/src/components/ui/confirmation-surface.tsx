import { useEffect, useId, useRef, type ReactNode } from 'react'

import { Button } from './button'
import type { ConfirmationDetail } from './types'

export interface ConfirmationSurfaceProps {
  title: string
  description: string
  details?: readonly ConfirmationDetail[]
  confirmLabel: string
  cancelLabel: string
  onConfirm: () => void
  onCancel: () => void
  destructive?: boolean
  autoFocus?: boolean
  children?: ReactNode
}

export function ConfirmationSurface({
  title,
  description,
  details = [],
  confirmLabel,
  cancelLabel,
  onConfirm,
  onCancel,
  destructive = false,
  autoFocus = false,
  children,
}: ConfirmationSurfaceProps) {
  const surfaceId = useId()
  const titleId = `${surfaceId}-confirmation-title`
  const descriptionId = `${surfaceId}-confirmation-description`
  const surfaceRef = useRef<HTMLElement>(null)
  const previousFocusRef = useRef<HTMLElement | null>(null)

  useEffect(() => {
    if (!autoFocus) return

    previousFocusRef.current = document.activeElement instanceof HTMLElement
      ? document.activeElement
      : null
    surfaceRef.current?.querySelector<HTMLButtonElement>('button:not([disabled])')?.focus()

    return () => {
      if (previousFocusRef.current?.isConnected) previousFocusRef.current.focus()
    }
  }, [autoFocus])

  return (
    <section
      ref={surfaceRef}
      aria-labelledby={titleId}
      aria-describedby={descriptionId}
      className="rounded-box border border-warning/60 bg-base-100 p-5 shadow-sm"
    >
      <h2 id={titleId} className="text-lg font-semibold">
        {title}
      </h2>
      <p id={descriptionId} className="mt-2 max-w-3xl text-sm text-base-content/75">
        {description}
      </p>
      {details.length > 0 ? (
        <dl className="mt-4 grid gap-2 text-sm sm:grid-cols-2">
          {details.map((detail) => (
            <div key={detail.label}>
              <dt className="font-medium text-base-content/70">{detail.label}</dt>
              <dd>{detail.value}</dd>
            </div>
          ))}
        </dl>
      ) : null}
      {children ? <div className="mt-4">{children}</div> : null}
      <div className="mt-5 flex flex-wrap justify-end gap-2">
        <Button variant="ghost" onClick={onCancel}>
          {cancelLabel}
        </Button>
        <Button variant={destructive ? 'danger' : 'primary'} onClick={onConfirm}>
          {confirmLabel}
        </Button>
      </div>
    </section>
  )
}
