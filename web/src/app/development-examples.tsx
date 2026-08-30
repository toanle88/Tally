import { useRef, useState } from 'react'

import {
  Button,
  ConfirmationSurface,
  DataTable,
  Field,
  Heading,
  Link,
  Panel,
  StatusBadge,
  type SemanticState,
} from '@/components/ui'
import { useScopeContext, type ScopeSnapshot } from '@/lib/scope/scope-context'

import { RecordDetailExample } from './record-detail-example'

const semanticStates: readonly SemanticState[] = [
  'success', 'warning', 'error', 'info', 'pending', 'reconciled', 'restricted', 'disabled',
]

const surfaceRows = [
  { surface: 'Status badge', purpose: 'Text meaning and accessible name' },
  { surface: 'Responsive shell', purpose: 'Stable application regions' },
  { surface: 'Semantic field', purpose: 'Label, help text, and error association' },
  { surface: 'Confirmation surface', purpose: 'Explicit action boundary and outcome context' },
]

const surfaceColumns = [
  { key: 'surface', header: 'Surface', render: (row: (typeof surfaceRows)[number]) => row.surface },
  { key: 'purpose', header: 'Purpose', render: (row: (typeof surfaceRows)[number]) => row.purpose },
] as const

export function DevelopmentExamples() {
  const { currentScope, hasUnsavedChanges, setHasUnsavedChanges, captureScopeSnapshot, runIfCurrentScope } = useScopeContext()
  const workSnapshot = useRef<ScopeSnapshot | null>(null)
  const [submissionMessage, setSubmissionMessage] = useState('No fixture submission has been attempted.')

  const startScopedWork = () => {
    workSnapshot.current = captureScopeSnapshot()
    setHasUnsavedChanges(true)
    setSubmissionMessage('Fixture work started against the selected accounting scope.')
  }

  const submitScopedWork = () => {
    const submitted = runIfCurrentScope(workSnapshot.current, () => {
      setHasUnsavedChanges(false)
      setSubmissionMessage('Fixture submission accepted for the current scope. No financial state changed.')
    })
    if (!submitted) setSubmissionMessage('Fixture submission blocked because its captured scope is stale.')
  }

  return (
    <>
      <div>
        <Heading level={2}>Development examples</Heading>
        <p className="mt-2 max-w-3xl text-base-content/75">
          Synthetic presentation examples only. Authentication, authorization, database readiness,
          finance capabilities, and authoritative financial records are not connected.
        </p>
      </div>

      <Panel title="Scope-bound example" description="Demonstrates explicit scope selection and stale-work prevention.">
        <div className="space-y-4">
          <p className="text-sm text-base-content/75">
            Current scope: <span className="font-medium">{currentScope?.legalEntity.name ?? 'None selected'}</span>
          </p>
          <div className="flex flex-wrap gap-3">
            <Button onClick={startScopedWork} disabled={hasUnsavedChanges || !currentScope}>Start scoped work</Button>
            <Button variant="secondary" onClick={submitScopedWork} disabled={!workSnapshot.current}>Submit scoped example</Button>
          </div>
          <p role="status" aria-live="polite" className="text-sm text-base-content/75">{submissionMessage}</p>
        </div>
      </Panel>

      <div className="grid items-start gap-6 xl:grid-cols-2">
        {(['finance-light', 'finance-dark'] as const).map((theme) => (
          <Panel key={theme} title={theme === 'finance-light' ? 'Light theme' : 'Dark theme'} description={`${theme} tokens and semantic states.`}>
            <div data-theme={theme} className="rounded-box bg-base-200 p-4">
              <div className="flex flex-wrap gap-2">{semanticStates.map((state) => <StatusBadge key={state} state={state} />)}</div>
            </div>
          </Panel>
        ))}
      </div>

      <div className="grid items-start gap-6 xl:grid-cols-2">
        <Panel title="Shared primitives" description="Common controls with semantic support.">
          <div className="space-y-5">
            <div className="flex flex-wrap items-center gap-3"><Button>Primary action</Button><Button variant="secondary">Secondary action</Button><Link href="#confirmation">View confirmation example</Link></div>
            <Field id="foundation-example" label="Example field" description="This field demonstrates label and help-text association." defaultValue="Example value" />
          </div>
        </Panel>
        <Panel title="Semantic table" description="Static accessible table primitive; advanced worklist behavior is deferred.">
          <DataTable caption="Foundation surfaces" columns={surfaceColumns} rows={surfaceRows} getRowKey={(row) => row.surface} />
        </Panel>
      </div>

      <div id="confirmation">
        <ConfirmationSurface
          title="Preview confirmation surface"
          description="This inline surface demonstrates a deliberate confirmation boundary without changing application state."
          details={[{ label: 'Record', value: 'Synthetic preview record' }, { label: 'Scope', value: currentScope?.id ?? 'No accounting scope selected' }, { label: 'Intended state change', value: 'Demonstrate confirmation only' }, { label: 'Financial effect', value: 'None — application state is unchanged' }]}
          confirmLabel="Confirm preview"
          cancelLabel="Cancel"
          onConfirm={() => undefined}
          onCancel={() => undefined}
        />
      </div>

      <RecordDetailExample />
    </>
  )
}
