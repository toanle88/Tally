import { fireEvent, render, screen } from '@testing-library/react'
import { useState } from 'react'
import { describe, expect, it, vi } from 'vitest'

import { CorrectionLineagePanel } from '@/components/correction-lineage-panel'
import { EvidenceDrawer } from '@/components/evidence-drawer'
import { LegalHoldIndicator } from '@/components/legal-hold-indicator'
import { LifecycleTimeline } from '@/components/lifecycle-timeline'
import { MoneyAndCurrencyPanel } from '@/components/money-and-currency-panel'
import { RecordIdentityHeader } from '@/components/record-identity-header'
import { SensitiveDataGuard } from '@/components/sensitive-data-guard'
import type { MoneyAmount } from '@/components/record-context'

import { evidenceFixture, lifecycleFixture, lineageReferences, moneyFixture, recordIdentityFixture } from '@/app/record-detail-fixtures'

describe('record context components', () => {
  it('renders identity, ownership, scope, lifecycle, version, and sensitivity context', () => {
    render(<RecordIdentityHeader identity={recordIdentityFixture} />)

    expect(screen.getByText('Customer receipt — RCPT-FIX-00042')).toBeInTheDocument()
    expect(screen.getByText('Accounts Receivable', { exact: true })).toBeInTheDocument()
    expect(screen.getByText(/Acme Vietnam Co\., Ltd\./)).toBeInTheDocument()
    expect(screen.getByText('v3')).toBeInTheDocument()
    expect(screen.getByText('Standard business record')).toBeInTheDocument()
    expect(screen.getByText('Established')).toBeInTheDocument()
  })

  it('orders lifecycle transitions chronologically and preserves decisions and links', () => {
    render(<LifecycleTimeline transitions={lifecycleFixture} />)

    const timeline = screen.getAllByRole('list')[0]
    const states = Array.from(timeline.querySelectorAll('li')).map((item) => item.textContent ?? '')
    expect(states[0]).toContain('Draft')
    expect(states[1]).toContain('Validated')
    expect(states[2]).toContain('Established')
    expect(screen.getByText('Authoritative receipt established')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /reversal: Receipt reversal fixture/ })).toBeInTheDocument()
  })

  it('keeps money roles and evidence labels distinct without calculating amounts', () => {
    render(<MoneyAndCurrencyPanel amounts={moneyFixture} gainLoss={{ amount: '-25.00', currency: 'USD', signConvention: 'Negative means loss; positive means gain.' }} />)

    expect(screen.getAllByText('1000.00 USD')).toHaveLength(2)
    expect(screen.getByText('24500000.00 VND')).toBeInTheDocument()
    expect(screen.getAllByText('Rounding')).toHaveLength(3)
    expect(screen.getAllByText(/FX-FIX-2026-08 · Spot/)).toHaveLength(2)
    expect(screen.getByText('Negative means loss; positive means gain.')).toBeInTheDocument()
  })

  it('does not accept or coerce numeric money values', () => {
    const numericAmount = { role: 'transaction', amount: 12.5, currency: 'USD', roundingLabel: 'Fixture' } as unknown as MoneyAmount & { role: 'transaction' }

    render(<MoneyAndCurrencyPanel amounts={{ transaction: numericAmount }} />)

    expect(screen.getByText(/numeric monetary values are not accepted/)).toBeInTheDocument()
    expect(screen.queryByText('12.5 USD')).not.toBeInTheDocument()
  })

  it('preserves the original fact and renders every correction lineage kind without edit controls', () => {
    render(<CorrectionLineagePanel original={{ recordId: recordIdentityFixture.recordId, label: 'Original established customer receipt', href: '#original', establishedAt: recordIdentityFixture.lastMaterialChange }} corrections={lineageReferences} />)

    expect(screen.getByRole('link', { name: /Original established customer receipt/ })).toBeInTheDocument()
    for (const reference of lineageReferences) expect(screen.getByRole('link', { name: new RegExp(reference.label) })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /edit|delete|destroy/i })).not.toBeInTheDocument()
  })

  it('shows available evidence as links and restricted or missing evidence safely', () => {
    function EvidenceHarness() {
      const [open, setOpen] = useState(false)
      return <EvidenceDrawer open={open} evidence={evidenceFixture} onOpenChange={setOpen} />
    }

    render(<EvidenceHarness />)
    fireEvent.click(screen.getByRole('button', { name: 'Open evidence drawer' }))

    expect(screen.getByRole('complementary', { name: 'Evidence drawer' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /Source receipt observation/ })).toBeInTheDocument()
    expect(screen.getByText(/Access restricted: Provider detail/)).toBeInTheDocument()
    expect(screen.getByText(/Evidence unavailable: Reconciliation evidence/)).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Close evidence drawer' }))
    expect(screen.queryByRole('complementary', { name: 'Evidence drawer' })).not.toBeInTheDocument()
  })

  it('masks restricted values, records denied actions, and permits explicit authorized actions', () => {
    const onAccessDenied = vi.fn()
    const onExport = vi.fn()
    render(<><SensitiveDataGuard label="Restricted value" classification="Bank-sensitive" value="SYNTHETIC-SECRET" access="restricted" canReveal={true} canExport={true} onAccessDenied={onAccessDenied} /><SensitiveDataGuard label="Authorized value" classification="Synthetic" value="SYNTHETIC-AUTHORIZED" access="authorized" canReveal={true} canExport={true} onExport={onExport} onAccessDenied={onAccessDenied} /></>)

    fireEvent.click(screen.getByRole('button', { name: 'Reveal unavailable' }))
    fireEvent.click(screen.getByRole('button', { name: 'Export unavailable' }))
    expect(onAccessDenied).toHaveBeenNthCalledWith(1, 'reveal')
    expect(onAccessDenied).toHaveBeenNthCalledWith(2, 'export')
    expect(screen.queryByText('SYNTHETIC-SECRET')).not.toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Reveal value' }))
    expect(screen.getByText('SYNTHETIC-AUTHORIZED')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Export value' }))
    expect(onExport).toHaveBeenCalledOnce()
  })

  it('blocks retention destruction under active legal hold independently of business lifecycle', () => {
    const onActivate = vi.fn()
    render(<LegalHoldIndicator status="active" holdReference="HOLD-FIX-00042" destructionAction={{ label: 'Destroy retained content', onActivate }} />)

    const button = screen.getByRole('button', { name: 'Destruction blocked by legal hold' })
    expect(button).toBeDisabled()
    expect(screen.getByText(/Business correction remains a separate/)).toBeInTheDocument()
    fireEvent.click(button)
    expect(onActivate).not.toHaveBeenCalled()
  })
})
