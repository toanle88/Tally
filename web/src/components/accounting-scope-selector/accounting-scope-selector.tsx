import { ConfirmationSurface } from '@/components/ui'
import { useScopeContext } from '@/lib/scope/scope-context'

export function AccountingScopeSelector() {
  const {
    currentScope,
    availableScopes,
    pendingScope,
    hasUnsavedChanges,
    requestScopeChange,
    confirmScopeChange,
    cancelScopeChange,
    lastChangeError,
  } = useScopeContext()

  return (
    <div className="space-y-4">
      <label className="block" htmlFor="accounting-scope-selector">
        <span className="label-text font-medium">Accounting scope</span>
        <select
          id="accounting-scope-selector"
          aria-label="Accounting scope"
          className="select select-bordered mt-2 min-h-11 w-full border-2 border-base-300 bg-base-100 text-base-content focus:border-primary focus:outline-2 focus:outline-offset-1 focus:outline-primary"
          value={currentScope?.id ?? ''}
          onChange={(event) => requestScopeChange(event.currentTarget.value)}
        >
          <option value="" disabled>
            Select an accounting scope
          </option>
          {availableScopes.map((scope) => (
            <option key={scope.id} value={scope.id}>
              {scope.legalEntity.name} — {scope.accountingBook.name}
            </option>
          ))}
        </select>
      </label>

      {currentScope ? (
        <dl className="grid gap-2 text-sm">
          <ScopeValue label="Tenant" value={currentScope.tenant.name} />
          <ScopeValue label="Legal entity" value={currentScope.legalEntity.name} />
          <ScopeValue label="Ledger" value={currentScope.ledger.name} />
          <ScopeValue label="Accounting book" value={currentScope.accountingBook.name} />
          <ScopeValue label="Functional currency" value={currentScope.functionalCurrency} />
          {currentScope.period ? <ScopeValue label="Period" value={currentScope.period.label} /> : null}
        </dl>
      ) : (
        <p className="text-sm text-base-content/70">No accounting scope is selected.</p>
      )}

      {lastChangeError ? (
        <p role="alert" className="text-sm text-error">
          {lastChangeError}
        </p>
      ) : null}

      {pendingScope ? (
        <ConfirmationSurface
          title="Unsaved work in the current scope"
          description={`Switching to ${pendingScope.legalEntity.name} will discard the unsaved fixture work.`}
          details={[
            { label: 'Current scope', value: currentScope?.legalEntity.name ?? 'None selected' },
            { label: 'Requested scope', value: pendingScope.legalEntity.name },
            { label: 'Unsaved changes', value: hasUnsavedChanges ? 'Yes' : 'No' },
          ]}
          confirmLabel="Discard and switch"
          cancelLabel="Return to current work"
          onConfirm={confirmScopeChange}
          onCancel={cancelScopeChange}
          destructive
        />
      ) : null}
    </div>
  )
}

function ScopeValue({ label, value }: { label: string; value: string }) {
  return (
    <div className="grid grid-cols-[minmax(0,8rem)_minmax(0,1fr)] gap-2">
      <dt className="font-medium text-base-content/70">{label}</dt>
      <dd>{value}</dd>
    </div>
  )
}
