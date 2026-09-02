import { useId } from 'react'

import { Button, Link } from '@/components/ui'
import type { EvidenceLink } from '@/components/record-context'

export interface EvidenceDrawerProps {
  open: boolean
  evidence: readonly EvidenceLink[]
  onOpenChange: (open: boolean) => void
}

const kindLabels: Record<EvidenceLink['kind'], string> = {
  source: 'Source',
  approval: 'Approval',
  posting: 'Posting',
  'provider-authority': 'Provider or authority',
  reconciliation: 'Reconciliation',
  close: 'Close',
  statement: 'Statement',
  audit: 'Audit',
}

export function EvidenceDrawer({ open, evidence, onOpenChange }: EvidenceDrawerProps) {
  const drawerId = `${useId()}-evidence-drawer`

  return (
    <section aria-label="Evidence">
      <Button variant="secondary" aria-expanded={open} aria-controls={drawerId} onClick={() => onOpenChange(!open)}>{open ? 'Close evidence drawer' : 'Open evidence drawer'}</Button>
      {open ? <aside id={drawerId} aria-label="Evidence drawer" className="mt-4 rounded-box border border-base-300 bg-base-100 p-4 shadow-sm"><h3 className="text-lg font-semibold">Evidence links</h3>{evidence.length === 0 ? <p className="mt-3 text-sm text-base-content/70">No evidence links supplied.</p> : <ul className="mt-3 space-y-2">{evidence.map((item) => <li key={item.id} className="rounded-box border border-base-300 p-3"><p className="text-sm font-medium">{kindLabels[item.kind]}</p>{item.access === 'available' && item.href ? <Link href={item.href}>{item.label} ({item.reference})</Link> : item.access === 'restricted' ? <p className="text-sm text-base-content/75">Access restricted: {item.restrictionReason ?? 'this evidence is not available to the current user.'}</p> : <p className="text-sm text-base-content/70">Evidence unavailable: {item.label} ({item.reference})</p>}</li>)}</ul>}</aside> : null}
    </section>
  )
}
