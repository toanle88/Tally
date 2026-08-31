import { Button, Link, Panel, StatusBadge } from '@/components/ui'
import type { ApiProblem, LookupStatus, ResultLookupOutcome, ResultLookupQuery } from '@/components/operational-context'

export interface ResultLookupProps {
  query: ResultLookupQuery
  status: LookupStatus
  outcome?: ResultLookupOutcome
  problem?: ApiProblem
  onLookup: (query: ResultLookupQuery) => void
}

export function ResultLookup({ query, status, outcome, problem, onLookup }: ResultLookupProps) {
  return (
    <Panel title="Established result lookup" description="Lookup uses business identity and request fingerprint; it never resubmits a mutation.">
      <dl className="mt-4 grid gap-3 sm:grid-cols-2"><div><dt className="text-sm font-medium text-base-content/70">Business identity</dt><dd className="mt-1 break-words font-mono">{query.businessIdentity}</dd></div><div><dt className="text-sm font-medium text-base-content/70">Request fingerprint</dt><dd className="mt-1 break-words font-mono">{query.requestFingerprint}</dd></div></dl>
      <div className="mt-4 flex flex-wrap items-center gap-3"><Button variant="secondary" loading={status === 'loading'} onClick={() => onLookup(query)}>Refresh result lookup</Button><span className="text-sm text-base-content/70">Lookup status: {status}</span></div>
      {status === 'unavailable' ? <p role="alert" className="mt-4 rounded-box border border-warning/40 bg-warning/5 p-3 text-sm">Result lookup is unavailable. Retry the status lookup or return to the worklist.</p> : null}
      {problem ? <p role="alert" className="mt-3 text-sm text-error">{problem.title}: {problem.detail} Correlation ID: {problem.correlationId}</p> : null}
      {status === 'found' && outcome ? <OutcomeCard outcome={outcome} /> : status === 'idle' ? <p className="mt-4 text-sm text-base-content/70">No established result has been looked up.</p> : null}
    </Panel>
  )
}

function OutcomeCard({ outcome }: { outcome: ResultLookupOutcome }) {
  const descriptions = {
    'safe-duplicate': 'Safe duplicate: the existing result is authoritative; do not resubmit.',
    'ambiguous-outcome': 'Ambiguous outcome: reconcile the established result before any deliberate retry.',
    'identity-content-conflict': 'Identity-content conflict: the identity was reused with different content.',
  } as const

  return <div className="mt-4 rounded-box border border-base-300 p-4"><div className="flex flex-wrap items-center gap-2"><StatusBadge state={outcome.status.semanticState} label={outcome.status.label} /><span className="text-sm font-medium">{descriptions[outcome.kind]}</span></div><p className="mt-3 text-sm"><span className="font-medium">Next action:</span> {outcome.nextAction}</p>{'establishedResult' in outcome && outcome.establishedResult ? <p className="mt-2"><Link href={outcome.establishedResult.href}>{outcome.establishedResult.label}</Link></p> : null}{outcome.kind === 'identity-content-conflict' ? <dl className="mt-3 grid gap-2 text-sm sm:grid-cols-2"><div><dt className="font-medium text-base-content/70">Established fingerprint</dt><dd className="break-words font-mono">{outcome.establishedFingerprint}</dd></div><div><dt className="font-medium text-base-content/70">Required new business identity</dt><dd className="mt-1"><Link href={outcome.requiredNewBusinessIdentity.href}>{outcome.requiredNewBusinessIdentity.label}</Link></dd></div></dl> : null}</div>
}
