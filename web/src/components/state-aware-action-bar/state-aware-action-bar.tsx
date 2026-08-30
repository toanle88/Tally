import { Button, Panel, StatusBadge } from '@/components/ui'
import type { ActionState, MaterialActionResult } from '@/components/operational-context'

export interface StateAwareActionBarProps {
  actions: readonly ActionState[]
  lastMaterialResult?: MaterialActionResult
  onAction: (action: ActionState) => void
}

export function StateAwareActionBar({ actions, lastMaterialResult, onAction }: StateAwareActionBarProps) {
  return (
    <Panel title="Available actions" description="Actions reflect the supplied current state; the server remains authoritative.">
      <div className="mt-4 space-y-3">
        <div className="flex flex-wrap gap-2">
          {actions.map((action) => (
            <div key={action.id} className="space-y-2">
              <Button
                variant={action.permitted ? 'primary' : 'ghost'}
                disabled={!action.permitted}
                aria-describedby={!action.permitted ? `${action.id}-blocked` : undefined}
                onClick={() => action.permitted ? onAction(action) : undefined}
              >
                {action.label}
              </Button>
              {!action.permitted ? (
                <div id={`${action.id}-blocked`} className="max-w-xs text-sm text-base-content/70">
                  <StatusBadge state="disabled" label="Blocked" />
                  <p className="mt-1">{action.blockingReason ?? 'This action is unavailable in the current state.'}</p>
                  {action.safeNextAction ? <p className="mt-1"><span className="font-medium">Next:</span> {action.safeNextAction}</p> : null}
                </div>
              ) : action.requiredConfirmation ? (
                <p className="max-w-xs text-sm text-base-content/70">Confirmation: {action.requiredConfirmation}</p>
              ) : null}
            </div>
          ))}
        </div>
        {actions.length === 0 ? <p className="text-sm text-base-content/70">No actions are available for this state.</p> : null}
        {lastMaterialResult ? (
          <p role="status" aria-live="polite" className="rounded-box border border-success/30 bg-success/5 p-3 text-sm">
            Actions refreshed after {lastMaterialResult.resultLabel}: {lastMaterialResult.stateLabel}. {lastMaterialResult.message}
          </p>
        ) : (
          <p className="text-sm text-base-content/70">No material result has refreshed these actions.</p>
        )}
      </div>
    </Panel>
  )
}
