import { useId, type ReactNode } from 'react'

import { classNames } from './class-names'
import type { SemanticState } from './types'

export interface PanelProps {
  title: string
  description?: string
  state?: SemanticState
  children: ReactNode
  className?: string
}

export function Panel({
  title,
  description,
  state,
  children,
  className,
}: PanelProps) {
  const titleId = `${useId()}-panel-title`

  return (
    <section
      aria-labelledby={titleId}
      data-state={state}
      className={classNames('card min-w-0 border border-base-300 bg-base-100 shadow-sm', className)}
    >
      <div className="card-body">
        <h2 id={titleId} className="card-title">
          {title}
        </h2>
        {description ? (
          <p className="flex-grow-0 text-sm text-base-content/70">{description}</p>
        ) : null}
        {children}
      </div>
    </section>
  )
}
