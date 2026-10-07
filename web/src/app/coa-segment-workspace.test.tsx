import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { developmentScopes } from './app'
import { CoaSegmentChangeRequest } from './coa-segment-change-request'
import { CoaSegmentCombinationValidator, CoaSegmentDefinitionRecord, CoaSegmentValueRecord, CoaSegmentWorklist } from './coa-segment-workspace'
import { coaMaintainSegmentDefinitions, coaMaintainSegmentValues, coaRequestSegmentChanges, coaValidateSegmentCombinations } from '@/generated/api/sdk.gen'
import { ScopeProvider } from '@/lib/scope/scope-context'

vi.mock('@/generated/api/sdk.gen', () => ({
  coaMaintainSegmentDefinitions: vi.fn(),
  coaMaintainSegmentValues: vi.fn(),
  coaRequestSegmentChanges: vi.fn(),
  coaValidateSegmentCombinations: vi.fn(),
}))

function renderWithScope(ui: ReactNode, initialEntry: string) {
  return render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <ScopeProvider availableScopes={developmentScopes} initialScopeId={developmentScopes[0].id} effects={{ cancelInFlightWork: vi.fn(), clearScopeBoundQueryState: vi.fn() }}>
        {ui}
      </ScopeProvider>
    </MemoryRouter>,
  )
}

describe('COA segment workspace', () => {
  beforeEach(() => {
    vi.mocked(coaMaintainSegmentDefinitions).mockReset()
    vi.mocked(coaMaintainSegmentValues).mockReset()
    vi.mocked(coaRequestSegmentChanges).mockReset()
    vi.mocked(coaValidateSegmentCombinations).mockReset()
  })

  it('shows the selected scope worklist and filters safe adapter records', () => {
    renderWithScope(<CoaSegmentWorklist />, '/coa-segments/coa-ws-01')

    expect(screen.getByRole('heading', { name: 'Segment administration worklist' })).toBeInTheDocument()
    expect(screen.getByRole('note')).toHaveTextContent('local safe adapter')
    expect(screen.getByRole('link', { name: 'Operations' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Validate combination' })).toHaveAttribute('href', '/coa-segments/coa-scr-03')
    expect(screen.getByText('2 records found.')).toBeInTheDocument()

    fireEvent.change(screen.getByRole('textbox', { name: 'Search segment definitions' }), { target: { value: 'shared' } })

    expect(screen.getByText('1 record found.')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Shared Services' })).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: 'Operations' })).not.toBeInTheDocument()
  })

  it('blocks invalid lifecycle transitions before calling the live command', () => {
    renderWithScope(<CoaSegmentDefinitionRecord />, '/coa-segments/coa-scr-01?segmentDefinitionId=segment-department-operations')

    fireEvent.change(screen.getByRole('combobox', { name: 'Lifecycle status' }), { target: { value: 'draft' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save segment-definition revision' }))

    expect(screen.getByRole('alert')).toHaveTextContent('The lifecycle transition from active to draft is not allowed.')
    expect(coaMaintainSegmentDefinitions).not.toHaveBeenCalled()
  })

  it('submits the live mutation and refreshes the safe adapter from its projection', async () => {
    vi.mocked(coaMaintainSegmentDefinitions).mockResolvedValue({
      data: {
        status: 'established',
        aggregateId: 'segment-department-finance',
        aggregateVersion: 1,
        correlationId: 'correlation-fixture',
        links: { self: '/api/v1/coa-segments/configuration/maintain-segment-definitions' },
        data: {
          segmentDefinition: {
            id: 'segment-department-finance',
            scopeId: 'scope-vietnam-statutory',
            segmentType: 'department',
            code: 'D-900',
            name: 'Finance',
            status: 'draft',
            effectiveDateFrom: '2026-01-01',
            version: 1,
            revisionNumber: 1,
            approvalStatus: 'not-required',
            validationOutcome: 'valid',
            nextAction: 'maintain',
          },
        },
      },
      error: undefined,
    } as never)

    renderWithScope(<CoaSegmentDefinitionRecord />, '/coa-segments/coa-scr-01?new=true')
    fireEvent.change(screen.getByRole('textbox', { name: 'Code' }), { target: { value: 'D-900' } })
    fireEvent.change(screen.getByRole('textbox', { name: 'Name' }), { target: { value: 'Finance' } })
    fireEvent.click(screen.getByRole('button', { name: 'Create segment definition' }))

    await waitFor(() => expect(coaMaintainSegmentDefinitions).toHaveBeenCalledTimes(1))
    expect(await screen.findByText(/Accepted segment-definition revision v1/)).toBeInTheDocument()
    expect(screen.getByDisplayValue('segment-department-finance')).toBeInTheDocument()
  })

  it('filters the worklist by value and exposes a value record link', () => {
    renderWithScope(<CoaSegmentWorklist />, '/coa-segments/coa-ws-01')

    fireEvent.change(screen.getByRole('textbox', { name: 'Search segment definitions' }), { target: { value: '1000' } })

    expect(screen.getByText('1 record found.')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: '1000' })).toHaveAttribute('href', expect.stringContaining('coa-scr-02'))
    expect(screen.getByRole('link', { name: 'Operations' })).toBeInTheDocument()
  })

  it('blocks a value outside the parent definition interval', () => {
    renderWithScope(<CoaSegmentValueRecord />, '/coa-segments/coa-scr-02?segmentDefinitionId=segment-department-shared-services&new=true')

    fireEvent.change(screen.getByRole('textbox', { name: 'Value' }), { target: { value: '3000' } })
    fireEvent.change(screen.getByRole('textbox', { name: 'Description' }), { target: { value: 'Future services' } })
    fireEvent.click(screen.getByRole('button', { name: 'Create segment value' }))

    expect(screen.getByRole('alert')).toHaveTextContent('must be contained within the parent definition interval')
    expect(coaMaintainSegmentValues).not.toHaveBeenCalled()
  })

  it('submits a value mutation and refreshes the safe value adapter', async () => {
    vi.mocked(coaMaintainSegmentValues).mockResolvedValue({
      data: {
        status: 'established',
        aggregateId: 'segment-department-operations',
        aggregateVersion: 4,
        correlationId: 'correlation-fixture',
        links: { self: '/api/v1/coa-segments/configuration/maintain-segment-values' },
        data: {
          segmentValue: {
            id: 'segment-value-operations-3000',
            segmentDefinitionId: 'segment-department-operations',
            scopeId: 'scope-vietnam-statutory',
            value: '3000',
            description: 'Finance operations',
            status: 'draft',
            effectiveDateFrom: '2026-01-01',
            version: 4,
            revisionNumber: 4,
            approvalStatus: 'not-required',
            validationOutcome: 'valid',
            nextAction: 'maintain',
          },
        },
      },
      error: undefined,
    } as never)

    renderWithScope(<CoaSegmentValueRecord />, '/coa-segments/coa-scr-02?segmentDefinitionId=segment-department-operations&new=true')
    fireEvent.change(screen.getByRole('textbox', { name: 'Value' }), { target: { value: '3000' } })
    fireEvent.change(screen.getByRole('textbox', { name: 'Description' }), { target: { value: 'Finance operations' } })
    fireEvent.click(screen.getByRole('button', { name: 'Create segment value' }))

    await waitFor(() => expect(coaMaintainSegmentValues).toHaveBeenCalledTimes(1))
    expect(await screen.findByText(/Accepted segment-value revision v4/)).toBeInTheDocument()
    expect(screen.getByDisplayValue('3000')).toBeInTheDocument()
  })

  it('validates a proposed combination and displays source versions without a mutation surface', async () => {
    vi.mocked(coaValidateSegmentCombinations).mockResolvedValue({
      data: {
        status: 'established',
        aggregateId: '00000000-0000-0000-0000-000000000000',
        aggregateVersion: 0,
        correlationId: 'correlation-fixture',
        links: { self: '/api/v1/coa-segments/actions/validate-segment-combinations' },
        data: {
          validationStatus: 'valid',
          effectiveDateResult: 'effective',
          sourceVersions: [{ segmentDefinitionId: 'segment-department-operations', segmentValueId: 'segment-value-operations-1000', segmentDefinitionVersion: 3, segmentDefinitionRevision: 3 }],
          invalidValues: [],
          restrictions: [],
          rejectionReasons: [],
          nextAction: 'proceed',
        },
      },
      error: undefined,
    } as never)

    renderWithScope(<CoaSegmentCombinationValidator />, '/coa-segments/coa-scr-03')
    expect(screen.getByRole('heading', { name: 'Segment combination validator' })).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Validate combination' }))

    await waitFor(() => expect(coaValidateSegmentCombinations).toHaveBeenCalledTimes(1))
    expect(await screen.findByDisplayValue('valid')).toBeInTheDocument()
    expect(screen.getByText(/segment-department-operations \/ segment-value-operations-1000: v3, revision 3/)).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /create|update|activate|suspend|approve/i })).not.toBeInTheDocument()
  })

  it('shows validation rejection reasons and keeps the request read-only', async () => {
    vi.mocked(coaValidateSegmentCombinations).mockResolvedValue({
      data: {
        status: 'established',
        aggregateId: '00000000-0000-0000-0000-000000000000',
        aggregateVersion: 0,
        correlationId: 'correlation-fixture',
        links: { self: '/api/v1/coa-segments/actions/validate-segment-combinations' },
        data: {
          validationStatus: 'invalid',
          effectiveDateResult: 'not-effective',
          sourceVersions: [{ segmentDefinitionId: 'segment-department-operations', segmentValueId: 'segment-value-operations-1000', segmentDefinitionVersion: 3, segmentDefinitionRevision: 3 }],
          invalidValues: [{ segmentDefinitionId: 'segment-department-operations', segmentValueId: 'segment-value-operations-1000', reason: 'the segment value is not active for validation' }],
          restrictions: ['lifecycle'],
          rejectionReasons: [{ segmentDefinitionId: 'segment-department-operations', segmentValueId: 'segment-value-operations-1000', reason: 'the segment value is not active for validation' }],
          nextAction: 'correct-and-revalidate',
        },
      },
      error: undefined,
    } as never)

    renderWithScope(<CoaSegmentCombinationValidator />, '/coa-segments/coa-scr-03')
    fireEvent.click(screen.getByRole('button', { name: 'Validate combination' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('the segment value is not active for validation')
    expect(screen.getByDisplayValue('invalid')).toBeInTheDocument()
    expect(screen.getByDisplayValue('correct-and-revalidate')).toBeInTheDocument()
  })

  it('requires a Workflow reference and shows the established request without changing the subject adapter', async () => {
    vi.mocked(coaRequestSegmentChanges).mockResolvedValue({
      data: {
        status: 'established',
        aggregateId: 'request-001',
        aggregateVersion: 1,
        correlationId: 'correlation-fixture',
        links: { self: '/api/v1/coa-segments/actions/request-segment-changes' },
        data: {
          segmentChangeRequest: {
            id: 'request-001',
            scopeId: 'scope-vietnam-statutory',
            changeType: 'definition',
            subjectId: 'segment-department-operations',
            subjectVersion: 3,
            requestedEffectiveDate: '2026-01-01',
            approvalRequestId: '11111111-1111-4111-8111-111111111111',
            approvalStatus: 'pending',
            applicationStatus: 'not-applied',
            validationOutcome: 'valid',
            nextAction: 'await-approval',
            proposedChange: { name: 'Operations and Shared Services' },
            version: 1,
            revisionNumber: 1,
          },
        },
      },
      error: undefined,
    } as never)

    renderWithScope(<CoaSegmentChangeRequest />, '/coa-segments/coa-scr-04?changeType=definition&subjectId=segment-department-operations')
    fireEvent.click(screen.getByRole('button', { name: 'Request definition change' }))
    expect(screen.getAllByRole('alert')[0]).toHaveTextContent('valid Workflow approval-request reference')
    expect(coaRequestSegmentChanges).not.toHaveBeenCalled()

    fireEvent.change(screen.getByRole('textbox', { name: 'Workflow approval-request reference' }), { target: { value: '11111111-1111-4111-8111-111111111111' } })
    fireEvent.change(screen.getByRole('textbox', { name: 'Name' }), { target: { value: 'Operations and Shared Services' } })
    fireEvent.click(screen.getByRole('button', { name: 'Request definition change' }))

    await waitFor(() => expect(coaRequestSegmentChanges).toHaveBeenCalledTimes(1))
    expect(await screen.findByDisplayValue('request-001')).toBeInTheDocument()
    expect(screen.getByDisplayValue('pending')).toBeInTheDocument()
    expect(screen.getByDisplayValue('not-applied')).toBeInTheDocument()
    expect(screen.getByRole('status')).toHaveTextContent('subject remains unchanged')
  })
})
