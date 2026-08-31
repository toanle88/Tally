import { Panel, StatusBadge } from '@/components/ui'
import type { ApprovalSnapshot } from '@/components/workflow-context'

export function ApprovalPanel({ approval }: { approval: ApprovalSnapshot }) {
  return <Panel title="Approval" description="Approval decision and application are separate lifecycle facts." state={approval.application.status === 'applied' ? 'success' : 'pending'}>
    <dl className="mt-4 grid gap-3 sm:grid-cols-2 text-sm">
      <div><dt className="font-medium text-base-content/70">Request</dt><dd>{approval.requestId}</dd></div>
      <div><dt className="font-medium text-base-content/70">Policy</dt><dd>{approval.policy.label} v{approval.policy.version}</dd></div>
      <div><dt className="font-medium text-base-content/70">Subject version</dt><dd>{approval.subject.version} ({approval.subject.snapshotLabel})</dd></div>
      <div><dt className="font-medium text-base-content/70">Scope</dt><dd>{approval.scope.label}</dd></div>
    </dl>
    <div className="mt-4 flex flex-wrap items-center gap-2"><span className="font-medium">Decision:</span><StatusBadge state={approval.status === 'rejected' ? 'error' : approval.status === 'applied' ? 'success' : 'pending'} label={approval.status} /></div>
    <ol className="mt-4 space-y-3 border-l-2 border-base-300 pl-4">
      {approval.steps.map((step) => <li key={step.id}><div className="flex flex-wrap items-center gap-2"><span className="font-medium">{step.label}</span><StatusBadge state={step.status === 'rejected' || step.status === 'expired' || step.status === 'invalidated' ? 'error' : step.status === 'applied' ? 'success' : 'pending'} label={step.status} /></div><p className="text-sm text-base-content/70">Owner: {step.owner.label}{step.decision ? ` — ${step.decision}` : ''}</p></li>)}
    </ol>
    <div className="mt-4 rounded-box border border-warning/40 bg-warning/5 p-3 text-sm"><strong>Application:</strong> {approval.application.status}. {approval.application.reason ?? `Requires subject version ${approval.application.requiredSubjectVersion} revalidation.`}</div>
  </Panel>
}
