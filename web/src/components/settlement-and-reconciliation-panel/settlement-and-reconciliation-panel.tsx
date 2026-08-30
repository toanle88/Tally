import { Panel, StatusBadge } from '@/components/ui'
import type { SettlementAmount, SettlementSnapshot } from '@/components/operational-context'

export interface SettlementAndReconciliationPanelProps {
  settlement: SettlementSnapshot
}

const balanceLabels: readonly [keyof Pick<SettlementSnapshot, 'gross' | 'returned' | 'reversed' | 'cancelled' | 'remaining' | 'net'>, string][] = [
  ['gross', 'Gross'],
  ['returned', 'Returned'],
  ['reversed', 'Reversed'],
  ['cancelled', 'Canceled'],
  ['remaining', 'Remaining'],
  ['net', 'Net'],
]

export function SettlementAndReconciliationPanel({ settlement }: SettlementAndReconciliationPanelProps) {
  return (
    <Panel title="Settlement and reconciliation" description="Balances and process states are supplied by the owning capability and are not calculated here.">
      <dl className="mt-4 grid gap-3 sm:grid-cols-2">
        {balanceLabels.map(([key, label]) => <BalanceValue key={key} label={label} amount={settlement[key]} />)}
        <StateValue label="Owner acknowledgement" state={settlement.ownerAcknowledgement} />
        <StateValue label="Reconciliation" state={settlement.reconciliation} />
      </dl>
      {settlement.exception ? (
        <div className="mt-4 rounded-box border border-warning/40 bg-warning/5 p-3">
          <p className="text-sm font-medium">Settlement exception</p>
          <div className="mt-2 flex flex-wrap items-center gap-2"><StatusBadge state={settlement.exception.semanticState} label={settlement.exception.label} />{settlement.exception.detail ? <span className="text-sm">{settlement.exception.detail}</span> : null}</div>
        </div>
      ) : null}
    </Panel>
  )
}

function BalanceValue({ label, amount }: { label: string; amount: SettlementAmount }) {
  return <div><dt className="text-sm font-medium text-base-content/70">{label}</dt><dd className="mt-1 font-mono">{amount.amount} {amount.currency}</dd></div>
}

function StateValue({ label, state }: { label: string; state: SettlementSnapshot['ownerAcknowledgement'] }) {
  return <div><dt className="text-sm font-medium text-base-content/70">{label}</dt><dd className="mt-1"><StatusBadge state={state.semanticState} label={state.label} /></dd></div>
}
