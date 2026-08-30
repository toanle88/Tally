export interface AccountingScopeFixture {
  id: string
  tenant: { id: string; name: string }
  legalEntity: { id: string; name: string }
  ledger: { id: string; name: string }
  accountingBook: { id: string; name: string }
  functionalCurrency: string
  period?: { id: string; label: string }
}

export type AuthScopeResolution =
  | {
      status: 'authenticated'
      actorLabel: string
      availableScopes: readonly AccountingScopeFixture[]
      initialScopeId: string | null
    }
  | {
      status: 'unauthenticated'
      reason: string
    }

export interface AuthScopeAdapter {
  resolve(): AuthScopeResolution
}
