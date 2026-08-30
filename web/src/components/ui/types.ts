import type { ReactNode } from 'react'

export type SemanticState =
  | 'success'
  | 'warning'
  | 'error'
  | 'info'
  | 'pending'
  | 'reconciled'
  | 'restricted'
  | 'disabled'

export type ButtonVariant =
  | 'primary'
  | 'secondary'
  | 'neutral'
  | 'ghost'
  | 'danger'

export type ControlSize = 'sm' | 'md' | 'lg'

export interface TableColumn<Row> {
  key: string
  header: string
  render: (row: Row) => ReactNode
}

export interface ConfirmationDetail {
  label: string
  value: ReactNode
}
