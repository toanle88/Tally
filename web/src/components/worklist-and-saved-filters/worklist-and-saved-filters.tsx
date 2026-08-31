import { useMemo, useState } from 'react'
import {
  createColumnHelper,
  flexRender,
  getCoreRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useReactTable,
  type ColumnDef,
  type PaginationState,
  type RowSelectionState,
  type SortingState,
  type VisibilityState,
} from '@tanstack/react-table'

import type { ApiProblem, ActionState, WorklistColumnId, WorklistExportContext, WorklistExportState, WorklistFilters, WorklistItem, WorklistStatus, SavedWorklistView } from '@/components/operational-context'
import { Button, Field, Link, Panel, Select, StatusBadge } from '@/components/ui'

export interface WorklistAndSavedFiltersProps {
  items: readonly WorklistItem[]
  status: WorklistStatus
  problem?: ApiProblem
  savedViews: readonly SavedWorklistView[]
  initialFilters?: Partial<WorklistFilters>
  initialVisibleColumns?: readonly WorklistColumnId[]
  pageSize?: number
  bulkAction?: ActionState
  onBulkAction?: (items: readonly WorklistItem[]) => void
  exportState?: WorklistExportState
  onExport?: (context: WorklistExportContext) => void
}

const allFilters: WorklistFilters = { scopeId: '', state: '', ownerId: '', date: '', amount: '', currency: '', exception: '', approval: '' }
const columnIds: readonly WorklistColumnId[] = ['record', 'scope', 'state', 'owner', 'age', 'amount', 'exception', 'approval', 'nextAction']
const columnHelper = createColumnHelper<WorklistItem>()

const statusLabels: Record<WorklistStatus, { label: string; semanticState: 'pending' | 'warning' | 'error' | 'success' | 'reconciled' }> = {
  loading: { label: 'Loading', semanticState: 'pending' },
  ready: { label: 'Ready', semanticState: 'success' },
  unavailable: { label: 'Unavailable', semanticState: 'warning' },
  rejected: { label: 'Rejected', semanticState: 'error' },
  partial: { label: 'Partial result', semanticState: 'warning' },
  reconciled: { label: 'Reconciled', semanticState: 'reconciled' },
}

export function WorklistAndSavedFilters({
  items,
  status,
  problem,
  savedViews,
  initialFilters,
  initialVisibleColumns = columnIds,
  pageSize = 3,
  bulkAction,
  onBulkAction,
  exportState = { permitted: false, blockedReason: 'Export is unavailable for this fixture.' },
  onExport,
}: WorklistAndSavedFiltersProps) {
  const [filters, setFilters] = useState<WorklistFilters>({ ...allFilters, ...initialFilters })
  const [selectedViewId, setSelectedViewId] = useState('')
  const [visibleColumns, setVisibleColumns] = useState<VisibilityState>(() => Object.fromEntries(columnIds.map((id) => [id, initialVisibleColumns.includes(id)])))
  const [sorting, setSorting] = useState<SortingState>([])
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize })
  const [rowSelection, setRowSelection] = useState<RowSelectionState>({})

  const filteredItems = useMemo(() => items.filter((item) => (
    (!filters.scopeId || item.scope.id === filters.scopeId)
    && (!filters.state || item.state.value === filters.state)
    && (!filters.ownerId || item.owner.id === filters.ownerId)
    && (!filters.date || item.date === filters.date)
    && (!filters.amount || item.amount?.amount === filters.amount)
    && (!filters.currency || item.amount?.currency === filters.currency)
    && (!filters.exception || item.exception?.value === filters.exception)
    && (!filters.approval || item.approval?.value === filters.approval)
  )), [filters, items])

  const columns = useMemo<ColumnDef<WorklistItem, string>[]>(() => [
    columnHelper.accessor((item) => item.record.label, { id: 'record', header: 'Record', cell: ({ row }) => <Link href={row.original.record.href}>{row.original.record.label}</Link> }),
    columnHelper.accessor((item) => item.scope.label, { id: 'scope', header: 'Scope', cell: ({ row }) => row.original.scope.label }),
    columnHelper.accessor((item) => item.state.label, { id: 'state', header: 'State', cell: ({ row }) => <StatusBadge state={row.original.state.semanticState} label={row.original.state.label} /> }),
    columnHelper.accessor((item) => item.owner.label, { id: 'owner', header: 'Owner', cell: ({ row }) => <span>{row.original.owner.label}{row.original.owner.reference ? ` (${row.original.owner.reference})` : ''}</span> }),
    columnHelper.accessor('age', { header: 'Age' }),
    columnHelper.accessor((item) => item.amount ? `${item.amount.amount} ${item.amount.currency}` : '', { id: 'amount', header: 'Amount / currency', cell: ({ row }) => row.original.amount ? <span className="font-mono">{row.original.amount.amount} {row.original.amount.currency}</span> : '—' }),
    columnHelper.accessor((item) => item.exception?.label ?? '', { id: 'exception', header: 'Exception', cell: ({ row }) => row.original.exception ? <StatusBadge state={row.original.exception.semanticState} label={row.original.exception.label} /> : '—' }),
    columnHelper.accessor((item) => item.approval?.label ?? '', { id: 'approval', header: 'Approval', cell: ({ row }) => row.original.approval ? <StatusBadge state={row.original.approval.semanticState} label={row.original.approval.label} /> : '—' }),
    columnHelper.accessor('nextAction', { header: 'Next action' }),
  ], [])

  const table = useReactTable({
    data: filteredItems,
    columns,
    state: { sorting, columnVisibility: visibleColumns, rowSelection, pagination },
    onSortingChange: setSorting,
    onColumnVisibilityChange: setVisibleColumns,
    onRowSelectionChange: setRowSelection,
    onPaginationChange: (updater) => {
      setPagination((current) => {
        const next = typeof updater === 'function' ? updater(current) : updater
        if (next.pageIndex !== current.pageIndex) setRowSelection({})
        return next
      })
    },
    enableRowSelection: (row) => row.original.bulkEligible,
    getRowId: (row) => row.id,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
  })

  const selectedItems = table.getSelectedRowModel().rows.map((row) => row.original)
  const bulkPermitted = Boolean(bulkAction?.permitted && selectedItems.length > 0 && selectedItems.every((item) => item.bulkEligible))
  const activeVisibleColumns = columnIds.filter((id) => visibleColumns[id] !== false)

  const updateFilter = (key: keyof WorklistFilters, value: string) => {
    setFilters((current) => ({ ...current, [key]: value }))
    setRowSelection({})
    setPagination((current) => ({ ...current, pageIndex: 0 }))
    setSelectedViewId('')
  }

  const applySavedView = (viewId: string) => {
    const view = savedViews.find((candidate) => candidate.id === viewId)
    setSelectedViewId(viewId)
    if (!view) return
    setFilters({ ...allFilters, ...view.filters })
    setVisibleColumns(Object.fromEntries(columnIds.map((id) => [id, view.visibleColumns.includes(id)])))
    setRowSelection({})
    setPagination((current) => ({ ...current, pageIndex: 0 }))
  }

  const exportContext: WorklistExportContext = { rows: filteredItems, filters, scopeId: filters.scopeId, visibleColumns: activeVisibleColumns }

  return (
    <Panel title="Worklist and saved filters" description="Fixture-backed worklist behavior with explicit scope, ownership, state, and eligibility context.">
      <div className="mt-4 space-y-5">
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <FilterSelect label="Scope" value={filters.scopeId} options={uniqueOptions(items.map((item) => ({ value: item.scope.id, label: item.scope.label })))} onChange={(value) => updateFilter('scopeId', value)} />
          <FilterSelect label="State" value={filters.state} options={uniqueOptions(items.map((item) => ({ value: item.state.value, label: item.state.label })))} onChange={(value) => updateFilter('state', value)} />
          <FilterSelect label="Owner" value={filters.ownerId} options={uniqueOptions(items.map((item) => ({ value: item.owner.id, label: item.owner.label })))} onChange={(value) => updateFilter('ownerId', value)} />
          <Field id="worklist-date-filter" label="Date" type="date" value={filters.date} onChange={(event) => updateFilter('date', event.target.value)} />
          <Field id="worklist-amount-filter" label="Amount" inputMode="decimal" value={filters.amount} onChange={(event) => updateFilter('amount', event.target.value)} placeholder="Exact decimal string" />
          <FilterSelect label="Currency" value={filters.currency} options={uniqueOptions(items.flatMap((item) => item.amount ? [{ value: item.amount.currency, label: item.amount.currency }] : []))} onChange={(value) => updateFilter('currency', value)} />
          <FilterSelect label="Exception" value={filters.exception} options={uniqueOptions(items.flatMap((item) => item.exception ? [{ value: item.exception.value, label: item.exception.label }] : []))} onChange={(value) => updateFilter('exception', value)} />
          <FilterSelect label="Approval" value={filters.approval} options={uniqueOptions(items.flatMap((item) => item.approval ? [{ value: item.approval.value, label: item.approval.label }] : []))} onChange={(value) => updateFilter('approval', value)} />
        </div>
        <div className="flex flex-wrap items-end gap-3">
          <FilterSelect label="Saved view" value={selectedViewId} options={savedViews.map((view) => ({ value: view.id, label: view.label }))} onChange={applySavedView} />
          <Button variant="ghost" onClick={() => { setFilters(allFilters); setSelectedViewId(''); setRowSelection({}); setPagination((current) => ({ ...current, pageIndex: 0 })) }}>Clear filters</Button>
          <Button variant="secondary" disabled={!exportState.permitted || !onExport} onClick={() => exportState.permitted && onExport ? onExport(exportContext) : undefined}>Export filtered worklist</Button>
          {!exportState.permitted ? <span className="max-w-sm text-sm text-base-content/70">{exportState.blockedReason ?? 'Export is unavailable.'}</span> : null}
        </div>
        <div className="flex flex-wrap gap-3" aria-label="Column visibility">
          {columnIds.map((id) => <label key={id} className="label cursor-pointer gap-2 py-1"><input className="checkbox checkbox-sm" type="checkbox" checked={visibleColumns[id] !== false} onChange={(event) => table.getColumn(id)?.toggleVisibility(event.target.checked)} /><span className="label-text capitalize">{id === 'nextAction' ? 'Next action' : id}</span></label>)}
        </div>
        <div className="flex flex-wrap items-center gap-3"><StatusBadge state={statusLabels[status].semanticState} label={statusLabels[status].label} announce /><span className="text-sm">Showing {filteredItems.length} of {items.length} fixture rows.</span></div>
        {problem ? <p role="alert" className="text-sm text-error">{problem.title}: {problem.detail} Correlation ID: {problem.correlationId}</p> : null}
        {status === 'loading' ? <p role="status" aria-busy="true" className="rounded-box border border-info/30 bg-info/5 p-4 text-sm">Worklist is loading. No result is implied.</p> : null}
        {status === 'unavailable' ? <p role="alert" className="rounded-box border border-warning/40 bg-warning/5 p-4 text-sm">Worklist dependency unavailable. Retry the lookup or return to the work area.</p> : null}
        {status === 'rejected' ? <p role="alert" className="rounded-box border border-error/40 bg-error/5 p-4 text-sm">The worklist request was rejected. Review the authoritative reason before changing filters.</p> : null}
        {status !== 'loading' && status !== 'unavailable' && status !== 'rejected' && filteredItems.length === 0 ? <p className="rounded-box border border-base-300 p-4 text-sm">No matching work items.</p> : null}
        {status !== 'loading' && status !== 'unavailable' && status !== 'rejected' && filteredItems.length > 0 ? <>
          <div className="overflow-x-auto"><table className="table min-w-full"><caption className="caption-top pb-3 text-left text-sm font-semibold">Operational work items</caption><thead className="bg-base-200/70">{table.getHeaderGroups().map((headerGroup) => <tr key={headerGroup.id}><th scope="col"><span className="sr-only">Select</span></th>{headerGroup.headers.map((header) => <th key={header.id} scope="col">{header.isPlaceholder ? null : <button type="button" className="font-semibold underline-offset-2 hover:underline" onClick={header.column.getToggleSortingHandler()}>{flexRender(header.column.columnDef.header, header.getContext())}{header.column.getIsSorted() ? ` (${header.column.getIsSorted() === 'asc' ? 'ascending' : 'descending'})` : ''}</button>}</th>)}</tr>)}</thead><tbody>{table.getRowModel().rows.map((row) => <tr key={row.id} data-selected={row.getIsSelected() ? 'true' : undefined} className={`border-b border-base-300/80 odd:bg-base-200/35 transition-colors hover:bg-info/10 ${row.getIsSelected() ? 'bg-primary/10' : ''}`}><td><input className="checkbox checkbox-sm" type="checkbox" aria-label={`Select ${row.original.record.label}`} checked={row.getIsSelected()} disabled={!row.original.bulkEligible} onChange={row.getToggleSelectedHandler()} /></td>{row.getVisibleCells().map((cell) => <td key={cell.id}>{flexRender(cell.column.columnDef.cell, cell.getContext())}</td>)}</tr>)}</tbody></table></div>
          <div className="flex flex-wrap items-center gap-3"><Button size="sm" variant="ghost" disabled={!table.getCanPreviousPage()} onClick={() => table.previousPage()}>Previous page</Button><span className="text-sm">Page {table.getState().pagination.pageIndex + 1} of {table.getPageCount()}</span><Button size="sm" variant="ghost" disabled={!table.getCanNextPage()} onClick={() => table.nextPage()}>Next page</Button></div>
          {bulkAction ? <div className="flex flex-wrap items-center gap-3"><Button variant="primary" disabled={!bulkPermitted} onClick={() => bulkPermitted && onBulkAction ? onBulkAction(selectedItems) : undefined}>{bulkAction.label}</Button><span className="text-sm text-base-content/70">Bulk actions require every selected row to be eligible; the server still validates each identity.{!bulkAction.permitted ? ` ${bulkAction.blockingReason ?? 'This action is blocked.'}` : ''}</span></div> : null}
        </> : null}
      </div>
    </Panel>
  )
}

function FilterSelect({ label, value, options, onChange }: { label: string; value: string; options: readonly { value: string; label: string }[]; onChange: (value: string) => void }) {
  return <div className="form-control min-w-48 flex-1 gap-2"><span className="label-text font-medium">{label}</span><Select aria-label={label} value={value} onChange={(event) => onChange(event.target.value)}><option value="">All</option>{options.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}</Select></div>
}

function uniqueOptions(options: readonly { value: string; label: string }[]) {
  return [...new Map(options.map((option) => [option.value, option])).values()]
}
