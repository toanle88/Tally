import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { invConfigureInvoiceTemplates, invGetInvoiceTemplate, invListInvoiceTemplates } from '@/generated/api/sdk.gen'
import { ScopeProvider } from '@/lib/scope/scope-context'

import { developmentScopes } from './app'
import { InvoiceTemplateEditor, InvoiceTemplateWorklist } from './invoice-template-workspace'

vi.mock('@/generated/api/sdk.gen', () => ({
  invConfigureInvoiceTemplates: vi.fn(),
  invGetInvoiceTemplate: vi.fn(),
  invListInvoiceTemplates: vi.fn(),
}))

const template = {
  id: '11111111-1111-4111-8111-111111111111',
  accountingScopeId: 'scope-vietnam-statutory',
  code: 'STANDARD-MONTHLY',
  name: 'Standard monthly service',
  billingFrequency: 'monthly',
  lineItems: [{ id: '22222222-2222-4222-8222-222222222222', code: 'SERVICE', description: 'Service charge', chargeRule: { kind: 'fixed', amount: '12.50', currency: 'USD' }, taxCategory: 'TAX-STANDARD' }],
  paymentMethod: { method: 'bank_transfer', termsDays: 30, paymentReference: 'provider-ref' },
  applicability: { mode: 'all' },
  effectiveFrom: '2026-01-01',
  version: 2,
  revisionNumber: 2,
  validationOutcome: 'valid',
  revisions: [{ revisionNumber: 1, capturedAt: '2026-01-01T00:00:00Z', fingerprint: 'sha256:one', version: 1 }, { revisionNumber: 2, capturedAt: '2026-02-01T00:00:00Z', fingerprint: 'sha256:two', version: 2 }],
}

function renderWithScope(ui: ReactNode, initialEntry: string) {
  return render(<MemoryRouter initialEntries={[initialEntry]}><ScopeProvider availableScopes={developmentScopes} initialScopeId={developmentScopes[0].id} effects={{ cancelInFlightWork: vi.fn(), clearScopeBoundQueryState: vi.fn() }}>{ui}</ScopeProvider></MemoryRouter>)
}

describe('invoice-template workspace', () => {
  beforeEach(() => {
    vi.mocked(invConfigureInvoiceTemplates).mockReset()
    vi.mocked(invGetInvoiceTemplate).mockReset()
    vi.mocked(invListInvoiceTemplates).mockReset()
  })

  it('loads the worklist from the authoritative query and links to the detail route', async () => {
    vi.mocked(invListInvoiceTemplates).mockResolvedValue({ data: { status: 'established', aggregateId: '00000000-0000-4000-8000-000000000000', aggregateVersion: 0, correlationId: '33333333-3333-4333-8333-333333333333', links: { self: '/api/v1/invoicing/reference/invoice-templates' }, data: { templates: [template], page: { size: 50, hasMore: false } } }, error: undefined } as never)
    renderWithScope(<InvoiceTemplateWorklist />, '/invoicing/inv-ws-01')

    expect(await screen.findByRole('link', { name: 'STANDARD-MONTHLY' })).toHaveAttribute('href', expect.stringContaining('invoiceTemplateId=11111111-1111-4111-8111-111111111111'))
    expect(screen.getByText('v2 / r2')).toBeInTheDocument()
    expect(invListInvoiceTemplates).toHaveBeenCalledWith(expect.objectContaining({ headers: { 'X-Accounting-Scope-Id': 'scope-vietnam-statutory' } }))
  })

  it('validates required fields and submits canonical exact-decimal values', async () => {
    vi.mocked(invConfigureInvoiceTemplates).mockResolvedValue({ data: { status: 'established', aggregateId: template.id, aggregateVersion: 1, correlationId: '33333333-3333-4333-8333-333333333333', links: { self: '/api/v1/invoicing/reference/invoice-templates/11111111-1111-4111-8111-111111111111' }, data: { template: { ...template, version: 1, revisionNumber: 1, revisions: [{ ...template.revisions[0] }], paymentMethod: { method: 'bank_transfer', termsDays: 0 }, lineItems: [{ ...template.lineItems[0], chargeRule: { kind: 'fixed', amount: '12.5', currency: 'USD' } }] }, replayed: false } }, error: undefined } as never)
    renderWithScope(<InvoiceTemplateEditor />, '/invoicing/inv-scr-01?new=true')

    fireEvent.click(screen.getByRole('button', { name: 'Create invoice template' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('Template code')
    expect(invConfigureInvoiceTemplates).not.toHaveBeenCalled()

    fireEvent.change(document.getElementById('invoice-template-code')!, { target: { value: ' standard-monthly ' } })
    fireEvent.change(document.getElementById('invoice-template-name')!, { target: { value: 'Standard monthly service' } })
    fireEvent.change(document.getElementById('invoice-template-effective-from')!, { target: { value: '2026-01-01' } })
    fireEvent.change(document.getElementById('invoice-template-line-0-code')!, { target: { value: ' SERVICE ' } })
    fireEvent.change(document.getElementById('invoice-template-line-0-description')!, { target: { value: 'Service charge' } })
    fireEvent.change(document.getElementById('invoice-template-line-0-amount')!, { target: { value: '00012.500' } })
    fireEvent.change(document.getElementById('invoice-template-line-0-tax-category')!, { target: { value: ' TAX-STANDARD ' } })
    fireEvent.click(screen.getByRole('button', { name: 'Create invoice template' }))

    await waitFor(() => expect(invConfigureInvoiceTemplates).toHaveBeenCalledTimes(1))
    const request = vi.mocked(invConfigureInvoiceTemplates).mock.calls[0][0]
    expect(request.body.data.code).toBe('standard-monthly')
    expect(request.body.data.lineItems[0].chargeRule).toEqual({ kind: 'fixed', amount: '12.5', currency: 'USD' })
    expect(request.body.data.lineItems[0].taxCategory).toBe('TAX-STANDARD')
    expect(request.headers['Idempotency-Key']).toBeTruthy()
  })

  it('shows an accessible version conflict and preserves the reviewed form', async () => {
    vi.mocked(invGetInvoiceTemplate).mockResolvedValue({ data: { status: 'established', aggregateId: template.id, aggregateVersion: template.version, correlationId: '33333333-3333-4333-8333-333333333333', links: { self: '/api/v1/invoicing/reference/invoice-templates/11111111-1111-4111-8111-111111111111' }, data: { template } }, error: undefined } as never)
    vi.mocked(invConfigureInvoiceTemplates).mockResolvedValue({ data: undefined, error: { status: 409, code: 'VERSION_CONFLICT', detail: 'The invoice template changed.', currentVersion: 3 } } as never)
    renderWithScope(<InvoiceTemplateEditor />, `/invoicing/inv-scr-01?invoiceTemplateId=${template.id}`)

    expect(await screen.findByDisplayValue('Standard monthly service')).toBeInTheDocument()
    fireEvent.change(screen.getByDisplayValue('Standard monthly service'), { target: { value: 'Reviewed replacement' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save invoice-template revision' }))

    expect(await screen.findByRole('dialog')).toHaveTextContent('record changed before this action')
    expect(screen.getByDisplayValue('Reviewed replacement')).toBeInTheDocument()
    expect(screen.getByText(/Refresh the authoritative template/)).toBeInTheDocument()
  })
})
