import { Panel, Link, StatusBadge } from '@/components/ui'
import type { LifecycleTransition } from '@/components/record-context'

export interface LifecycleTimelineProps {
  transitions: readonly LifecycleTransition[]
}
export function LifecycleTimeline({ transitions }: LifecycleTimelineProps) {
  const orderedTransitions = transitions
    .map((transition, index) => ({ transition, index }))
    .sort((left, right) => {
      const leftTime = Date.parse(left.transition.occurredAt)
      const rightTime = Date.parse(right.transition.occurredAt)
      const leftOrder = Number.isNaN(leftTime) ? Number.MAX_SAFE_INTEGER : leftTime
      const rightOrder = Number.isNaN(rightTime) ? Number.MAX_SAFE_INTEGER : rightTime
      return leftOrder - rightOrder || left.transition.sequence - right.transition.sequence || left.index - right.index
    })

  return (
    <Panel title="Lifecycle timeline" description="Established state transitions and linked correction paths.">
      {orderedTransitions.length === 0 ? (
        <p className="mt-4 text-sm text-base-content/70">No established state transitions recorded.</p>
      ) : (
        <ol className="mt-5 space-y-5 border-l-2 border-base-300 pl-5">
          {orderedTransitions.map(({ transition }) => (
            <li key={transition.id} className="relative">
              <span className="absolute -left-[1.65rem] top-1 h-4 w-4 rounded-full border-2 border-base-100 bg-primary" aria-hidden="true" />
              <div className="flex flex-wrap items-center gap-2"><StatusBadge state={transition.semanticState} label={transition.state} /><span className="text-sm text-base-content/70">by {transition.actor}</span></div>
              <time className="mt-1 block text-sm text-base-content/70" dateTime={transition.occurredAt}>{transition.occurredAt}</time>
              {transition.decision ? <p className="mt-2 text-sm"><span className="font-medium">Decision:</span> {transition.decision}</p> : null}
              {transition.references.length > 0 ? (
                <ul className="mt-2 space-y-1 text-sm">
                  {transition.references.map((reference) => <li key={`${transition.id}-${reference.recordId}`}><Link href={reference.href}>{reference.kind}: {reference.label}</Link></li>)}
                </ul>
              ) : null}
            </li>
          ))}
        </ol>
      )}
    </Panel>
  )
}
