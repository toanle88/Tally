import { Panel } from '@/components/ui'
import type { MoneyAmount, MoneyByRole, SignedGainLoss } from '@/components/record-context'

export interface MoneyAndCurrencyPanelProps {
  amounts: MoneyByRole
  gainLoss?: SignedGainLoss
}

const roles = ['transaction', 'functional', 'presentation'] as const

export function MoneyAndCurrencyPanel({ amounts, gainLoss }: MoneyAndCurrencyPanelProps) {
  return (
    <Panel title="Money and currency" description="Amounts retain their supplied financial meaning; no calculation occurs in the UI.">
      <div className="mt-4 space-y-4">
        {roles.map((role) => {
          const amount = amounts[role]
          return amount ? <MoneyRow key={role} amount={amount} /> : null
        })}
        {gainLoss ? <GainLossRow gainLoss={gainLoss} /> : null}
        {roles.every((role) => !amounts[role]) && !gainLoss ? <p className="text-sm text-base-content/70">No monetary values supplied.</p> : null}
      </div>
    </Panel>
  )
}

function MoneyRow({ amount }: { amount: MoneyAmount }) {
  const rawAmount = amount as unknown as { amount: unknown; currency: unknown; roundingLabel: unknown; rateEvidence?: MoneyAmount['rateEvidence'] }
  const isDisplayable = typeof rawAmount.amount === 'string' && typeof rawAmount.currency === 'string' && typeof rawAmount.roundingLabel === 'string'
  if (!isDisplayable) return <div className="rounded-box border border-error/40 p-3 text-sm text-error">Amount unavailable: numeric monetary values are not accepted.</div>

  const displayAmount = rawAmount.amount as string
  const displayCurrency = rawAmount.currency as string
  const displayRounding = rawAmount.roundingLabel as string

  return (
    <dl className="grid gap-1 rounded-box border border-base-300 p-3 sm:grid-cols-[10rem_1fr]" data-role={amount.role}>
      <dt className="font-medium capitalize">{amount.role} amount</dt>
      <dd className="font-mono">{displayAmount} {displayCurrency}</dd>
      <dt className="text-sm text-base-content/70">Rounding</dt>
      <dd className="text-sm">{displayRounding}</dd>
      {rawAmount.rateEvidence ? <><dt className="text-sm text-base-content/70">Rate evidence</dt><dd className="text-sm">{rawAmount.rateEvidence.rateSet} · {rawAmount.rateEvidence.rateType} · {rawAmount.rateEvidence.asOf} · {rawAmount.rateEvidence.sourceReference}{rawAmount.rateEvidence.rateValue ? ` · ${rawAmount.rateEvidence.rateValue}` : ''}</dd></> : null}
    </dl>
  )
}

function GainLossRow({ gainLoss }: { gainLoss: SignedGainLoss }) {
  const rawAmount = gainLoss as unknown as { amount: unknown; currency: unknown }
  return <dl className="grid gap-1 rounded-box border border-base-300 p-3 sm:grid-cols-[10rem_1fr]"><dt className="font-medium">Signed gain/loss</dt><dd className="font-mono">{typeof rawAmount.amount === 'string' && typeof rawAmount.currency === 'string' ? `${rawAmount.amount} ${rawAmount.currency}` : 'Amount unavailable'}</dd><dt className="text-sm text-base-content/70">Sign convention</dt><dd className="text-sm">{gainLoss.signConvention}</dd></dl>
}
