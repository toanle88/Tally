import { useMemo } from 'react'
import { RouterProvider } from 'react-router-dom'

import { createDefaultAuthClient, type AuthClient } from '@/lib/auth/auth-client'
import type { AuthScopeAdapter, AuthScopeResolution } from '@/lib/auth/auth-scope-adapter'
import { useAuthSession } from '@/lib/auth/auth-session-store'
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

const defaultScopeChangeEffects: ScopeChangeEffects = {
  cancelInFlightWork: () => undefined,
  clearScopeBoundQueryState: () => undefined,
}

export interface AppProps {
  authScopeAdapter?: AuthScopeAdapter
  authClient?: AuthClient
  scopeChangeEffects?: ScopeChangeEffects
}

function AuthStatus({ resolution, authClient, warning }: { resolution: AuthScopeResolution; authClient: AuthClient; warning: boolean }) {
  if (resolution.status === 'authenticated' && !warning) return null
  if (resolution.status === 'authenticated') {
    return <p role="status" aria-live="polite" className="mx-auto max-w-7xl border-b border-warning/40 bg-warning/10 px-4 py-3 text-sm">Your session expires soon. Save work before signing in again.</p>
  }
  if (resolution.status === 'loading') {
    return <p role="status" aria-live="polite" className="sr-only">Checking authentication.</p>
  }
  return <div role="alert" className="mx-auto max-w-7xl border-b border-warning/40 bg-warning/10 px-4 py-3 text-sm"><p>{resolution.reason}</p><button type="button" className="btn btn-sm btn-primary mt-2" onClick={() => void authClient.login()}>Sign in</button></div>
}

function App({ authScopeAdapter, authClient, scopeChangeEffects = defaultScopeChangeEffects }: AppProps) {
  const selectedAuthClient = useMemo(() => authClient ?? createDefaultAuthClient(developmentScopes), [authClient])
  const { resolution, expiryWarning } = useAuthSession(authScopeAdapter, selectedAuthClient)
  const router = useMemo(() => createAppRouter(), [])

  const availableScopes = resolution.status === 'authenticated' ? resolution.availableScopes : []
  const initialScopeId = resolution.status === 'authenticated' ? resolution.initialScopeId : null

  return (
    <AuthScopeProvider resolution={resolution} authClient={selectedAuthClient}>
      <ScopeProvider availableScopes={availableScopes} initialScopeId={initialScopeId} effects={scopeChangeEffects}>
        <AuthStatus resolution={resolution} authClient={selectedAuthClient} warning={expiryWarning} />
        <RouterProvider router={router} />
      </ScopeProvider>
    </AuthScopeProvider>
  )
}

export default App
