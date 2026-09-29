import { useState } from 'react'

import { Button, Panel, StatusBadge, type SemanticState } from '@/components/ui'

type PublicationStatus = 'pending_approval' | 'approved' | 'published' | 'rejected' | 'stale' | 'unavailable'

type PublicationRecord = {
  id: string
  aggregateType: string
  aggregateId: string
  version: number
  status: PublicationStatus
  effectiveFrom?: string
  effectiveTo?: string
  eventType?: string
  dependentAvailability: 'pending' | 'available' | 'unavailable'
  nextAction: string
}

const initialPublicationRecords: PublicationRecord[] = [
  { id: 'legal-entity-v4', aggregateType: 'Legal entity', aggregateId: 'entity-vietnam', version: 4, status: 'approved', effectiveFrom: '2026-01-01', eventType: 'LegalEntityPublished', dependentAvailability: 'pending', nextAction: 'Publish approved version' },
  { id: 'party-v3', aggregateType: 'Party', aggregateId: 'party-industrial', version: 3, status: 'pending_approval', effectiveFrom: '2026-01-01', dependentAvailability: 'pending', nextAction: 'Wait for approval' },
  { id: 'customer-v2', aggregateType: 'Customer profile', aggregateId: 'customer-industrial', version: 2, status: 'published', effectiveFrom: '2026-01-01', eventType: 'CustomerProfilePublished', dependentAvailability: 'available', nextAction: 'View publication evidence' },
  { id: 'vendor-v3', aggregateType: 'Vendor profile', aggregateId: 'vendor-industrial', version: 3, status: 'stale', effectiveFrom: '2026-01-01', dependentAvailability: 'pending', nextAction: 'Refresh approved version' },
  { id: 'fiscal-v1', aggregateType: 'Fiscal calendar', aggregateId: 'calendar-vietnam', version: 1, status: 'unavailable', effectiveFrom: '2026-01-01', eventType: 'FiscalCalendarPublished', dependentAvailability: 'unavailable', nextAction: 'Investigate availability' },
  { id: 'legacy-v8', aggregateType: 'Legal entity', aggregateId: 'entity-legacy', version: 8, status: 'rejected', effectiveFrom: '2020-01-01', effectiveTo: '2025-12-31', dependentAvailability: 'unavailable', nextAction: 'Review rejection evidence' },
]

const statusState: Record<PublicationStatus, SemanticState> = {
  pending_approval: 'pending',
  approved: 'success',
  published: 'reconciled',
  rejected: 'error',
  stale: 'warning',
  unavailable: 'disabled',
}

const statusLabel: Record<PublicationStatus, string> = {
  pending_approval: 'pending approval',
  approved: 'approved',
  published: 'published',
  rejected: 'rejected',
  stale: 'stale',
  unavailable: 'unavailable',
}

export function MasterDataPublicationReview() {
  const [records, setRecords] = useState(initialPublicationRecords)
  const [notice, setNotice] = useState('Publication is an explicit, auditable action. Downstream availability is eventual and does not create downstream master-data records here.')

  const publish = (record: PublicationRecord) => {
    if (record.status !== 'approved') {
      setNotice(`${record.aggregateType} ${record.aggregateId} cannot be published because its current version is ${statusLabel[record.status]}.`)
      return
    }
    setRecords((current) => current.map((candidate) => candidate.id === record.id ? { ...candidate, status: 'published', eventType: eventTypeFor(record.aggregateType), dependentAvailability: 'pending', nextAction: 'Monitor dependent availability' } : candidate))
    setNotice(`${record.aggregateType} ${record.aggregateId} version ${record.version} was published. The ${eventTypeFor(record.aggregateType)} outbox message is committed; dependent availability remains pending until consumers acknowledge it.`)
  }

  return <section aria-labelledby="omd-publication-review-title" className="space-y-6">
    <div>
      <p className="text-sm font-semibold uppercase tracking-wide text-primary">OMD-SCR-05</p>
      <h2 id="omd-publication-review-title" className="mt-1 text-2xl font-semibold">Approved master-data publication review</h2>
      <p className="mt-2 max-w-3xl text-base-content/75">Review the current approved aggregate version before making it available to dependent bounded contexts. Publication preserves the source version and effective dates; consumers establish their own local snapshots.</p>
    </div>
    <p role="status" aria-live="polite" className="rounded-box border border-info/30 bg-info/5 p-4 text-sm">{notice}</p>
    <Panel title="Publication lifecycle" description="Only approved records expose the publish action. Rejected, stale, and unavailable records expose their next safe action instead.">
      <div className="overflow-x-auto" tabIndex={0}>
        <table className="table table-zebra" aria-label="Master-data publication records">
          <thead className="text-base-content"><tr><th scope="col">Aggregate</th><th scope="col">Source version</th><th scope="col">State</th><th scope="col">Effective interval</th><th scope="col">Dependent availability</th><th scope="col">Next action</th></tr></thead>
          <tbody>{records.map((record) => <tr key={record.id}>
            <th scope="row"><span className="block font-semibold">{record.aggregateType}</span><span className="font-mono text-xs text-base-content/70">{record.aggregateId}</span></th>
            <td>v{record.version}</td>
            <td><StatusBadge state={statusState[record.status]} label={statusLabel[record.status]} /></td>
            <td>{record.effectiveFrom ? `${record.effectiveFrom} — ${record.effectiveTo ?? 'open'}` : 'not date-effective'}</td>
            <td><StatusBadge state={record.dependentAvailability === 'available' ? 'success' : record.dependentAvailability === 'unavailable' ? 'disabled' : 'pending'} label={record.dependentAvailability} /></td>
            <td><div className="flex min-w-48 flex-col items-start gap-2"><span className="text-sm">{record.nextAction}</span>{record.status === 'approved' ? <Button size="sm" onClick={() => publish(record)}>Publish approved version</Button> : null}{record.eventType ? <span className="font-mono text-xs text-base-content/70">{record.eventType} v1</span> : null}</div></td>
          </tr>)}</tbody>
        </table>
      </div>
    </Panel>
    <Panel title="Consumer boundary" description="The publication payload contains identifiers, aggregate version, approval evidence, effective dates, and a safe immutable snapshot. No downstream OMD write is performed from this screen.">
      <ul className="list-disc space-y-2 pl-5 text-sm text-base-content/75"><li>Published means the OMD publication record and transactional outbox message are committed.</li><li>Available means a dependent context has established its own local snapshot; pending is expected immediately after publication.</li><li>Retrying the same command is safe and does not create a second publication for the same aggregate version.</li></ul>
    </Panel>
  </section>
}

function eventTypeFor(aggregateType: string) {
  switch (aggregateType) {
    case 'Legal entity': return 'LegalEntityPublished'
    case 'Party': return 'PartyPublished'
    case 'Customer profile': return 'CustomerProfilePublished'
    case 'Vendor profile': return 'VendorProfilePublished'
    case 'Fiscal calendar': return 'FiscalCalendarPublished'
    default: return 'MasterDataPublished'
  }
}
