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
      status: 'loading'
    }
  | {
      status: 'authenticated'
      actorLabel: string
      availableScopes: readonly AccountingScopeFixture[]
      initialScopeId: string | null
      sessionClass?: 'general' | 'privileged'
      expiresAt?: Date
      warningAt?: Date
    }
  | {
      status: 'unauthenticated'
      reason: string
    }
  | {
      status: 'expired'
      reason: string
    }
  | {
      status: 'step-up-required'
      reason: string
    }

export interface AuthScopeAdapter {
  resolve(): AuthScopeResolution
}
