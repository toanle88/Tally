import { Button, Link, Panel, StatusBadge } from '@/components/ui'
import type { PostingSnapshot } from '@/components/workflow-context'

export function PostingPanel({ posting, onRetry }: { posting: PostingSnapshot; onRetry?: () => void }) {
  const state = posting.status === 'established-result' ? 'success' : posting.status === 'rejected' || posting.status === 'failure' ? 'error' : 'pending'
  return <Panel title="Posting" description="Posting admission and established results remain authoritative to the owning capability." state={state}>
    <div className="flex flex-wrap items-center gap-2"><StatusBadge state={state} label={posting.status} /><span className="text-sm text-base-content/70">Request {posting.requestReference}</span></div>
    <dl className="mt-4 grid gap-3 text-sm sm:grid-cols-2">
      <div><dt className="font-medium text-base-content/70">Source</dt><dd>{posting.sourceRecord.label} v{posting.sourceVersion}</dd></div>
      <div><dt className="font-medium text-base-content/70">Accounting owner</dt><dd>{posting.accountingOwner}</dd></div>
      <div><dt className="font-medium text-base-content/70">Scope</dt><dd>{posting.scope.label}</dd></div>
      {posting.establishedResult ? <div><dt className="font-medium text-base-content/70">Established result</dt><dd><Link href={posting.establishedResult.href}>{posting.establishedResult.label}</Link></dd></div> : null}
    </dl>
    {posting.periodGateEvidence ? <div className="mt-4 rounded-box border border-base-300 p-3 text-sm"><h3 className="font-medium">Period and gate evidence</h3><p className="mt-1">Period {posting.periodGateEvidence.fiscalPeriodId} · state version {posting.periodGateEvidence.periodStateVersion} · gate version {posting.periodGateEvidence.postingGateVersion} · {posting.periodGateEvidence.admission}</p></div> : null}
    {posting.failure ? <div className="mt-4 rounded-box border border-error/40 bg-error/5 p-3 text-sm"><p>{posting.failure.detail}</p><p className="mt-1">{posting.failure.nextAction}</p>{posting.failure.retryPermitted ? <Button className="mt-3" variant="secondary" onClick={onRetry}>Retry deliberately</Button> : <p className="mt-2 font-medium">Retry is unavailable.</p>}</div> : null}
    {posting.reconciliation ? <p className="mt-4 text-sm">Reconciliation: <strong>{posting.reconciliation.state}</strong>{posting.reconciliation.evidence ? <> — <Link href={posting.reconciliation.evidence.href}>{posting.reconciliation.evidence.label}</Link></> : null}</p> : null}
  </Panel>
}
