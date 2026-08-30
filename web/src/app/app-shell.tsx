import type { ReactNode } from 'react'

import { Heading } from '@/components/ui'

export interface AppShellProps {
  navigation: ReactNode
  scopeContext: ReactNode
  statusFeedback: ReactNode
  children: ReactNode
}

export function AppShell({
  navigation,
  scopeContext,
  statusFeedback,
  children,
}: AppShellProps) {
  return (
    <div
      data-testid="app-shell"
      data-theme="finance-light"
      className="min-h-svh bg-base-200 text-base-content"
    >
      <header className="border-b border-base-300 bg-base-100">
        <div className="mx-auto flex max-w-7xl items-center justify-between gap-4 px-4 py-5 sm:px-6 lg:px-8">
          <div>
            <Heading level={1}>TALLY</Heading>
            <p className="mt-1 text-sm text-base-content/70">
              Shared finance interface foundation
            </p>
          </div>
          <span className="hidden rounded-full bg-base-200 px-3 py-1 text-xs font-medium sm:inline-flex">
            Local preview
          </span>
        </div>
      </header>

      <div className="mx-auto grid max-w-7xl gap-6 px-4 py-6 sm:px-6 lg:grid-cols-[15rem_minmax(0,1fr)] lg:px-8">
        <aside className="space-y-6">
          <nav
            aria-label="Global navigation"
            className="rounded-box border border-base-300 bg-base-100 p-4 shadow-sm"
          >
            <Heading level={2} className="text-base">
              Navigation
            </Heading>
            <div className="mt-3">{navigation}</div>
          </nav>
          <section
            aria-labelledby="accounting-scope-context-title"
            className="rounded-box border border-base-300 bg-base-100 p-4 shadow-sm"
          >
            <h2 id="accounting-scope-context-title" className="text-base font-semibold">
              Accounting scope context
            </h2>
            <div className="mt-3 text-sm">{scopeContext}</div>
          </section>
        </aside>

        <div className="min-w-0 space-y-5">
          <section
            aria-label="Status feedback"
            aria-live="polite"
            className="rounded-box border border-info/30 bg-info/5 p-4 shadow-sm"
          >
            {statusFeedback}
          </section>
          <main id="main-content" aria-label="Application content" className="space-y-6">
            {children}
          </main>
        </div>
      </div>
    </div>
  )
}
