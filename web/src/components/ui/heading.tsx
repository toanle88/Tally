import type { ReactNode } from 'react'

export type HeadingLevel = 1 | 2 | 3 | 4 | 5 | 6

export interface HeadingProps {
  level: HeadingLevel
  children: ReactNode
  className?: string
}

const headingTags = {
  1: 'h1',
  2: 'h2',
  3: 'h3',
  4: 'h4',
  5: 'h5',
  6: 'h6',
} as const

const headingClasses = {
  1: 'text-3xl font-bold tracking-tight',
  2: 'text-2xl font-semibold tracking-tight',
  3: 'text-xl font-semibold',
  4: 'text-lg font-semibold',
  5: 'text-base font-semibold',
  6: 'text-sm font-semibold uppercase tracking-wide',
} as const

export function Heading({ level, children, className }: HeadingProps) {
  const Tag = headingTags[level]

  return <Tag className={[headingClasses[level], className].filter(Boolean).join(' ')}>{children}</Tag>
}
