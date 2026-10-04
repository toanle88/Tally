import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { developmentScopes } from './app'
import { CoaSegmentDefinitionRecord, CoaSegmentValueRecord, CoaSegmentWorklist } from './coa-segment-workspace'
import { coaMaintainSegmentDefinitions, coaMaintainSegmentValues } from '@/generated/api/sdk.gen'
import { ScopeProvider } from '@/lib/scope/scope-context'

vi.mock('@/generated/api/sdk.gen', () => ({
  coaMaintainSegmentDefinitions: vi.fn(),
  coaMaintainSegmentValues: vi.fn(),
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
  })

  it('shows the selected scope worklist and filters safe adapter records', () => {
    renderWithScope(<CoaSegmentWorklist />, '/coa-segments/coa-ws-01')

    expect(screen.getByRole('heading', { name: 'Segment administration worklist' })).toBeInTheDocument()
    expect(screen.getByRole('note')).toHaveTextContent('local safe adapter')
    expect(screen.getByRole('link', { name: 'Operations' })).toBeInTheDocument()
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
})
