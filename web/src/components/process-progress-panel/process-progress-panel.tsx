import { Link, Panel, StatusBadge } from '@/components/ui'
import type { ProcessState, ProcessStep, ProcessStepStatus } from '@/components/operational-context'

export interface ProcessProgressPanelProps {
  process: ProcessState
}

const statusLabels: Record<ProcessStepStatus, { label: string; semanticState: 'info' | 'pending' | 'error' | 'warning' | 'reconciled' | 'success' }> = {
  current: { label: 'Current', semanticState: 'info' },
  pending: { label: 'Pending', semanticState: 'pending' },
  failed: { label: 'Failed', semanticState: 'error' },
  'partially-completed': { label: 'Partially completed', semanticState: 'warning' },
  reconciled: { label: 'Reconciled', semanticState: 'reconciled' },
  terminal: { label: 'Terminal', semanticState: 'success' },
}

export function ProcessProgressPanel({ process }: ProcessProgressPanelProps) {
  return (
    <Panel title="Process progress" description="Cross-capability progress remains visible by owner; downstream outcomes are eventual, not synchronous success.">
      <div className="mt-4 flex flex-wrap items-center gap-3"><StatusBadge state={process.currentState.semanticState} label={process.currentState.label} /><span className="text-sm">Age: {process.age}</span></div>
      <ol className="mt-5 space-y-4 border-l-2 border-base-300 pl-5">{process.steps.map((step) => <ProcessStepRow key={step.id} step={step} />)}</ol>
      {process.exception ? <p className="mt-4 rounded-box border border-warning/40 bg-warning/5 p-3 text-sm">Process exception: <StatusBadge state={process.exception.semanticState} label={process.exception.label} /> {process.exception.detail ?? 'Resolution remains with the owning capability.'}</p> : null}
    </Panel>
  )
}

function ProcessStepRow({ step }: { step: ProcessStep }) {
  const status = statusLabels[step.status]
  return <li><div className="flex flex-wrap items-center gap-2"><StatusBadge state={status.semanticState} label={status.label} /><span className="font-medium">{step.label}</span></div><p className="mt-1 text-sm">Owner: {step.owner.label}{step.owner.reference ? ` (${step.owner.reference})` : ''}</p>{step.detail ? <p className="mt-1 text-sm text-base-content/70">{step.detail}</p> : null}{step.authoritativeRecord ? <p className="mt-1 text-sm"><Link href={step.authoritativeRecord.href}>{step.authoritativeRecord.label}</Link></p> : null}</li>
}
