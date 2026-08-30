import type { AnchorHTMLAttributes, ReactNode } from 'react'

import { classNames } from './class-names'

export type LinkTone = 'default' | 'muted' | 'danger'

export interface LinkProps
  extends Omit<AnchorHTMLAttributes<HTMLAnchorElement>, 'href'> {
  href: string
  tone?: LinkTone
  children: ReactNode
}

const toneClasses: Record<LinkTone, string> = {
  default: 'link link-primary focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary',
  muted: 'link link-neutral focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary',
  danger: 'link link-error focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-error',
}

export function Link({
  tone = 'default',
  className,
  children,
  ...rest
}: LinkProps) {
  return (
    <a {...rest} className={classNames(toneClasses[tone], className)}>
      {children}
    </a>
  )
}
