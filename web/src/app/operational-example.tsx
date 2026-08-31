import { useState } from 'react'

import { ExceptionResolutionPanel } from '@/components/exception-resolution-panel'
import { ProcessProgressPanel } from '@/components/process-progress-panel'
import { ResultLookup } from '@/components/result-lookup'
import { SettlementAndReconciliationPanel } from '@/components/settlement-and-reconciliation-panel'
import { StateAwareActionBar } from '@/components/state-aware-action-bar'
import { WorklistAndSavedFilters } from '@/components/worklist-and-saved-filters'
import { Button, Heading, Panel } from '@/components/ui'
import type { ExceptionResolutionResult, LookupStatus, MaterialActionResult, ResultLookupOutcome } from '@/components/operational-context'

import { exceptionFixture, operationalActions, operationalSavedViews, operationalWorklistFixture, processFixture, resultLookupFixtures, settlementFixture } from './operational-fixtures'

export function OperationalExample() {
  const [lastMaterialResult, setLastMaterialResult] = useState<MaterialActionResult>()
  const [resolutionResult, setResolutionResult] = useState<ExceptionResolutionResult>()
  const [lookupIndex, setLookupIndex] = useState(0)
  const [lookupStatus, setLookupStatus] = useState<LookupStatus>('found')
  const [message, setMessage] = useState('No operational fixture action has been attempted.')
  const result = resultLookupFixtures[lookupIndex] as ResultLookupOutcome

  const handleAction = (action: (typeof operationalActions)[number]) => {
    setMessage(`Fixture action selected: ${action.label}. No financial state changed.`)
    if (action.material) setLastMaterialResult({ actionId: action.id, resultLabel: action.label, stateLabel: 'Pending owner result', message: 'The fixture host refreshed the action state from a supplied result.', refreshedAt: '2026-08-30T11:45:00Z' })
  }

  return <section aria-labelledby="operational-example-title" className="space-y-6"><div><Heading level={2}>Operational worklist and process example</Heading><p id="operational-example-title" className="mt-2 max-w-3xl text-base-content/75">Synthetic presentation-only worklist, action, settlement, exception, result, and process surfaces. No callback establishes a finance fact.</p></div><WorklistAndSavedFilters items={operationalWorklistFixture} status="ready" savedViews={operationalSavedViews} initialFilters={{ scopeId: 'scope-vietnam-statutory' }} pageSize={2} bulkAction={{ id: 'assign', label: 'Assign selected work', permitted: true, material: false }} onBulkAction={(items) => setMessage(`Fixture bulk selection accepted for ${items.length} eligible row(s). Server validation remains authoritative.`)} exportState={{ permitted: true }} onExport={(context) => setMessage(`Fixture export prepared for ${context.rows.length} filtered row(s); no file was downloaded.`)} /><div id="settlement-example" className="grid items-start gap-6 xl:grid-cols-2"><StateAwareActionBar actions={operationalActions} lastMaterialResult={lastMaterialResult} onAction={handleAction} /><SettlementAndReconciliationPanel settlement={settlementFixture} /></div><div className="grid items-start gap-6 xl:grid-cols-2"><ExceptionResolutionPanel exception={exceptionFixture} resolutionResult={resolutionResult} onResolve={(resolution) => setResolutionResult({ resolutionId: resolution.id, state: { value: 'resolved', label: 'Exception resolved', semanticState: 'success' }, message: `Fixture resolution selected: ${resolution.label}.` })} /><Panel title="Result lookup outcome" description="Switch between the required duplicate, ambiguous, and conflict result categories."><div className="flex flex-wrap gap-2">{resultLookupFixtures.map((candidate, index) => <Button key={candidate.kind} variant="ghost" size="sm" aria-pressed={lookupIndex === index} onClick={() => { setLookupIndex(index); setLookupStatus('found') }}>{candidate.kind}</Button>)}</div><div className="mt-4"><ResultLookup query={result.query} status={lookupStatus} outcome={result} onLookup={() => { setLookupStatus('found'); setMessage('Fixture result lookup refreshed; no mutation was retried.') }} /></div><p id="new-business-identity" className="mt-3 text-sm text-base-content/70">Fixture destination for a required new business identity; the owning capability would provide the authoritative creation surface.</p></Panel></div><div id="process-example"><ProcessProgressPanel process={processFixture} /></div><p role="status" aria-live="polite" className="rounded-box border border-info/30 bg-info/5 p-3 text-sm">{message}</p></section>
}
