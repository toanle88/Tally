import { Button, Link, Panel, StatusBadge } from '@/components/ui'
import type { ActionState, ExceptionResolutionResult, ExceptionSnapshot } from '@/components/operational-context'

export interface ExceptionResolutionPanelProps {
  exception: ExceptionSnapshot
  resolutionResult?: ExceptionResolutionResult
  onResolve: (resolution: ActionState) => void
}

export function ExceptionResolutionPanel({ exception, resolutionResult, onResolve }: ExceptionResolutionPanelProps) {
  return (
    <Panel title="Exception resolution" description="Only authorized resolutions are actionable; the owning capability establishes the resulting state.">
      <dl className="mt-4 grid gap-3 sm:grid-cols-2">
        <div><dt className="text-sm font-medium text-base-content/70">Exception type</dt><dd className="mt-1">{exception.type}</dd></div>
        <div><dt className="text-sm font-medium text-base-content/70">Scope</dt><dd className="mt-1">{exception.scope.label}</dd></div>
        {exception.amount ? <div><dt className="text-sm font-medium text-base-content/70">Amount</dt><dd className="mt-1 font-mono">{exception.amount.amount} {exception.amount.currency}</dd></div> : null}
        <div><dt className="text-sm font-medium text-base-content/70">Owner</dt><dd className="mt-1">{exception.owner.label}{exception.owner.reference ? ` (${exception.owner.reference})` : ''}</dd></div>
        <div><dt className="text-sm font-medium text-base-content/70">Age</dt><dd className="mt-1">{exception.age}</dd></div>
        <div><dt className="text-sm font-medium text-base-content/70">Resulting state</dt><dd className="mt-1"><StatusBadge state={exception.resultingState.semanticState} label={exception.resultingState.label} /></dd></div>
      </dl>
      <div className="mt-4">
        <h3 className="font-medium">Evidence</h3>
        {exception.evidence.length === 0 ? <p className="mt-2 text-sm text-base-content/70">No evidence supplied.</p> : <ul className="mt-2 space-y-1 text-sm">{exception.evidence.map((item) => <li key={item.id}>{item.access === 'available' && item.href ? <Link href={item.href}>{item.label} ({item.reference})</Link> : item.access === 'restricted' ? <span className="text-warning-content">Evidence restricted: {item.restrictionReason ?? item.label}</span> : <span className="text-base-content/70">Evidence unavailable: {item.label} ({item.reference})</span>}</li>)}</ul>}
      </div>
      <div className="mt-4">
        <h3 className="font-medium">Authorized resolutions</h3>
        <div className="mt-2 flex flex-wrap gap-2">{exception.authorizedResolutions.map((resolution) => <ButtonForResolution key={resolution.id} resolution={resolution} onResolve={onResolve} />)}</div>
      </div>
      {resolutionResult ? <p role="status" aria-live="polite" className="mt-4 rounded-box border border-info/30 bg-info/5 p-3 text-sm">{resolutionResult.message} Resulting state: {resolutionResult.state.label}.</p> : null}
    </Panel>
  )
}

function ButtonForResolution({ resolution, onResolve }: { resolution: ActionState; onResolve: (resolution: ActionState) => void }) {
  const reasonId = `${resolution.id}-reason`
  return <div className="space-y-1"><Button variant="secondary" disabled={!resolution.permitted} aria-describedby={!resolution.permitted ? reasonId : undefined} onClick={() => resolution.permitted ? onResolve(resolution) : undefined}>{resolution.label}{!resolution.permitted ? <span className="sr-only"> unavailable</span> : null}</Button>{!resolution.permitted ? <p id={reasonId} className="max-w-xs text-sm text-base-content/70">{resolution.blockingReason ?? 'Resolution unavailable.'}{resolution.safeNextAction ? ` Next: ${resolution.safeNextAction}` : null}</p> : null}</div>
}
