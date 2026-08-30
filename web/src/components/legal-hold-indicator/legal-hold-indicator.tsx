import { Panel, StatusBadge, Button } from '@/components/ui'
import type { LegalHoldStatus } from '@/components/record-context'

export interface LegalHoldIndicatorProps {
  status: LegalHoldStatus
  holdReference?: string
  appliedAt?: string
  destructionAction?: { label: string; onActivate: () => void }
}
const statusLabels: Record<LegalHoldStatus, string> = { active: 'Active legal hold', released: 'Legal hold released', none: 'No active legal hold' }

export function LegalHoldIndicator({ status, holdReference, appliedAt, destructionAction }: LegalHoldIndicatorProps) {
  const active = status === 'active'
  return (
    <Panel title="Legal hold" description="Retention state is independent from business lifecycle state.">
      <div className="mt-4 space-y-3"><StatusBadge state={active ? 'warning' : status === 'released' ? 'success' : 'info'} label={statusLabels[status]} />{holdReference ? <p className="text-sm">Reference: {holdReference}</p> : null}{appliedAt ? <p className="text-sm">Applied: <time dateTime={appliedAt}>{appliedAt}</time></p> : null}{destructionAction ? <Button variant="danger" disabled={active} onClick={destructionAction.onActivate}>{active ? 'Destruction blocked by legal hold' : destructionAction.label}</Button> : null}<p className="text-sm text-base-content/70">Business correction remains a separate lifecycle and lineage concern.</p></div>
    </Panel>
  )
}
