import { AppShell } from './app-shell'
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

const semanticStates: readonly SemanticState[] = [
  'success',
  'warning',
  'error',
  'info',
  'pending',
  'reconciled',
  'restricted',
  'disabled',
]

const surfaceRows = [
  { surface: 'Status badge', purpose: 'Text meaning and accessible name' },
  { surface: 'Responsive shell', purpose: 'Stable application regions' },
  { surface: 'Semantic field', purpose: 'Label, help text, and error association' },
  { surface: 'Confirmation surface', purpose: 'Explicit action boundary and outcome context' },
]

const surfaceColumns = [
  {
    key: 'surface',
    header: 'Surface',
    render: (row: (typeof surfaceRows)[number]) => row.surface,
  },
  {
    key: 'purpose',
    header: 'Purpose',
    render: (row: (typeof surfaceRows)[number]) => row.purpose,
  },
] as const

function PreviewContent() {
  return (
    <>
      <div>
        <Heading level={2}>Design-system foundation</Heading>
        <p className="mt-2 max-w-3xl text-base-content/75">
          Synthetic presentation examples only. Authentication, authorization, database
          readiness, accounting scope, and finance capabilities are not connected.
        </p>
      </div>

      <div className="grid items-start gap-6 xl:grid-cols-2">
        {(['finance-light', 'finance-dark'] as const).map((theme) => (
          <Panel
            key={theme}
            title={theme === 'finance-light' ? 'Light theme' : 'Dark theme'}
            description={`${theme} tokens and semantic states.`}
          >
            <div data-theme={theme} className="rounded-box bg-base-200 p-4">
              <div className="flex flex-wrap gap-2">
                {semanticStates.map((state) => (
                  <StatusBadge key={state} state={state} />
                ))}
              </div>
            </div>
          </Panel>
        ))}
      </div>

      <div className="grid items-start gap-6 xl:grid-cols-2">
        <Panel title="Shared primitives" description="Common controls with semantic support.">
          <div className="space-y-5">
            <div className="flex flex-wrap items-center gap-3">
              <Button>Primary action</Button>
              <Button variant="secondary">Secondary action</Button>
              <Link href="#confirmation">View confirmation example</Link>
            </div>
            <Field
              id="foundation-example"
              label="Example field"
              description="This field demonstrates label and help-text association."
              defaultValue="Example value"
            />
          </div>
        </Panel>

        <Panel title="Semantic table" description="Static accessible table primitive; advanced worklist behavior is deferred.">
          <DataTable
            caption="Foundation surfaces"
            columns={surfaceColumns}
            rows={surfaceRows}
            getRowKey={(row) => row.surface}
          />
        </Panel>
      </div>

      <div id="confirmation">
        <ConfirmationSurface
          title="Preview confirmation surface"
          description="This inline surface demonstrates a deliberate confirmation boundary without changing application state."
          details={[
            { label: 'Record', value: 'Synthetic preview record' },
            { label: 'Scope', value: 'No accounting scope selected' },
            { label: 'Intended state change', value: 'Demonstrate confirmation only' },
            { label: 'Financial effect', value: 'None — application state is unchanged' },
          ]}
          confirmLabel="Confirm preview"
          cancelLabel="Cancel"
          onConfirm={() => undefined}
          onCancel={() => undefined}
        />
      </div>
    </>
  )
}

function App() {
  return (
    <AppShell
      navigation={<p className="text-sm text-base-content/70">Routed navigation is provided by a later delivery.</p>}
      scopeContext={<p className="text-base-content/70">No accounting scope is selected in this foundation preview.</p>}
      statusFeedback={<StatusBadge state="info" label="Foundation preview only" announce />}
    >
      <PreviewContent />
    </AppShell>
  )
}

export default App
