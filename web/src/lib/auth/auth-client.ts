import type { AccountingScopeFixture, AuthScopeResolution } from './auth-scope-adapter'
import { createMsalAuthClient } from './msal-auth-client'
import { withSessionDeadline } from './session'

export interface AuthClient {
  readonly initialResolution?: AuthScopeResolution
  initialize(): Promise<AuthScopeResolution>
  login(): Promise<void>
  logout(): Promise<void>
  getAccessToken(): Promise<string>
  refreshAccessToken(): Promise<string>
}

export interface FixtureAuthClientOptions {
  actorLabel: string
  availableScopes: readonly AccountingScopeFixture[]
  initialScopeId: string | null
}

export function createFixtureAuthClient(options: FixtureAuthClientOptions): AuthClient {
  const resolution = withSessionDeadline({
    status: 'authenticated',
    actorLabel: options.actorLabel,
    availableScopes: options.availableScopes,
    initialScopeId: options.initialScopeId,
    sessionClass: 'general',
  })
  return {
    initialResolution: resolution,
    initialize: async () => resolution,
    login: async () => undefined,
    logout: async () => undefined,
    getAccessToken: async () => 'local-signed-fixture-token',
    refreshAccessToken: async () => 'local-signed-fixture-token',
  }
}

export function createUnavailableAuthClient(): AuthClient {
  const resolution: AuthScopeResolution = {
    status: 'unauthenticated',
    reason: 'Sign-in is not configured for this environment.',
  }
  return {
    initialResolution: resolution,
    initialize: async () => resolution,
    login: async () => undefined,
    logout: async () => undefined,
    getAccessToken: async () => {
      throw new Error('authentication is unavailable')
    },
    refreshAccessToken: async () => {
      throw new Error('authentication is unavailable')
    },
  }
}

export function createDefaultAuthClient(availableScopes: readonly AccountingScopeFixture[]): AuthClient {
  const appEnvironment = import.meta.env.VITE_APP_ENV || 'local'
  const mode = import.meta.env.VITE_AUTH_MODE
  if (mode === 'fixture' && import.meta.env.DEV && appEnvironment === 'local') {
    return createFixtureAuthClient({
      actorLabel: 'Local signed-fixture user',
      availableScopes,
      initialScopeId: availableScopes[0]?.id ?? null,
    })
  }

  const clientId = import.meta.env.VITE_ENTRA_CLIENT_ID
  const authority = import.meta.env.VITE_ENTRA_AUTHORITY
  const apiScope = import.meta.env.VITE_ENTRA_API_SCOPE
  if (mode === 'msal' && clientId && authority && apiScope) {
    return createMsalAuthClient({
      clientId,
      authority,
      apiScope,
      redirectUri: import.meta.env.VITE_ENTRA_REDIRECT_URI,
    })
  }
  return createUnavailableAuthClient()
}
