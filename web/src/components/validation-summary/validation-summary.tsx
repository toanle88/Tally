import { useEffect, useId, useRef } from 'react'

import { Link, Panel } from '@/components/ui'
import type { ValidationIssue } from '@/components/workflow-context'

export interface ValidationSummaryProps {
  issues: readonly ValidationIssue[]
  onFocusTarget?: (targetId: string) => void
}

export function ValidationSummary({ issues, onFocusTarget }: ValidationSummaryProps) {
  const titleId = `${useId()}-validation-title`
  const summaryRef = useRef<HTMLDivElement>(null)
  useEffect(() => { if (issues.length > 0) summaryRef.current?.focus() }, [issues.length])
  if (issues.length === 0) return null
  return (
    <Panel title="Review the highlighted issues" description="Submission was not sent. Resolve each issue, then try again." className="border-error/50" state="error">
      <div ref={summaryRef} id={titleId} tabIndex={-1} className="mt-3" role="alert">
        <ul className="space-y-2 text-sm">
          {issues.map((issue) => (
            <li key={issue.id}>
              <Link href={`#${issue.targetId}`} onClick={(event) => { event.preventDefault(); onFocusTarget?.(issue.targetId) }}>
                <span className="font-medium">{issue.targetLabel}:</span> {issue.message}
              </Link>
              <span className="ml-2 text-base-content/60">({issue.category})</span>
              {issue.nextAction ? <p className="ml-4 text-base-content/70">Next: {issue.nextAction}</p> : null}
            </li>
          ))}
        </ul>
      </div>
    </Panel>
  )
}
