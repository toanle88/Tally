import { useMemo } from 'react'
import { RouterProvider } from 'react-router-dom'

import type { AuthScopeAdapter } from '@/lib/auth/auth-scope-adapter'
import { AuthScopeProvider, createAppRouter } from '@/routes/router'
import { ScopeProvider, type ScopeChangeEffects } from '@/lib/scope/scope-context'

export const developmentScopes = [
  {
    id: 'scope-vietnam-statutory',
    tenant: { id: 'tenant-acme', name: 'Acme Holdings (fixture)' },
    legalEntity: { id: 'entity-vietnam', name: 'Acme Vietnam Co., Ltd. (fixture)' },
    ledger: { id: 'ledger-vietnam', name: 'Primary Ledger' },
    accountingBook: { id: 'book-vietnam', name: 'Statutory Book' },
    functionalCurrency: 'VND',
    period: { id: 'period-2026-08', label: '2026-08 — Open' },
  },
  {
    id: 'scope-singapore-management',
    tenant: { id: 'tenant-acme', name: 'Acme Holdings (fixture)' },
    legalEntity: { id: 'entity-singapore', name: 'Acme Singapore Pte. Ltd. (fixture)' },
    ledger: { id: 'ledger-singapore', name: 'Regional Ledger' },
    accountingBook: { id: 'book-singapore', name: 'Management Book' },
    functionalCurrency: 'SGD',
    period: { id: 'period-2026-08', label: '2026-08 — Open' },
  },
] as const

const defaultAuthScopeAdapter: AuthScopeAdapter = {
  resolve: () => ({
    status: 'authenticated',
    actorLabel: 'Development user',
    availableScopes: developmentScopes,
    initialScopeId: developmentScopes[0].id,
  }),
}

const defaultScopeChangeEffects: ScopeChangeEffects = {
  cancelInFlightWork: () => undefined,
  clearScopeBoundQueryState: () => undefined,
}

export interface AppProps {
  authScopeAdapter?: AuthScopeAdapter
  scopeChangeEffects?: ScopeChangeEffects
}

function App({ authScopeAdapter = defaultAuthScopeAdapter, scopeChangeEffects = defaultScopeChangeEffects }: AppProps) {
  const resolution = useMemo(() => authScopeAdapter.resolve(), [authScopeAdapter])
  const router = useMemo(() => createAppRouter(), [])
  const availableScopes = resolution.status === 'authenticated' ? resolution.availableScopes : []
  const initialScopeId = resolution.status === 'authenticated' ? resolution.initialScopeId : null

  return (
    <AuthScopeProvider resolution={resolution}>
      <ScopeProvider availableScopes={availableScopes} initialScopeId={initialScopeId} effects={scopeChangeEffects}>
        <RouterProvider router={router} />
      </ScopeProvider>
    </AuthScopeProvider>
  )
}

export default App
