import { classNames } from './class-names'
import type { TableColumn } from './types'

export interface DataTableProps<Row> {
  caption: string
  columns: readonly TableColumn<Row>[]
  rows: readonly Row[]
  getRowKey: (row: Row, index: number) => string
  emptyMessage?: string
  className?: string
}

export function DataTable<Row>({
  caption,
  columns,
  rows,
  getRowKey,
  emptyMessage = 'No rows to display.',
  className,
}: DataTableProps<Row>) {
  return (
    <div className="overflow-x-auto">
      <table className={classNames('table min-w-full', className)}>
        <caption className="caption-top pb-3 text-left text-sm font-semibold">
          {caption}
        </caption>
        <thead className="bg-base-200/70">
          <tr>
            {columns.map((column) => (
              <th key={column.key} className="text-base-content" scope="col">
                {column.header}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.length > 0 ? (
            rows.map((row, index) => (
              <tr
                key={getRowKey(row, index)}
                className="border-b border-base-300/80 odd:bg-base-200/35 transition-colors hover:bg-info/10"
              >
                {columns.map((column) => (
                  <td key={column.key}>{column.render(row)}</td>
                ))}
              </tr>
            ))
          ) : (
            <tr>
              <td colSpan={Math.max(columns.length, 1)}>{emptyMessage}</td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  )
}
