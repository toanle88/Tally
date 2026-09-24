import {
  InteractionRequiredAuthError,
  PublicClientApplication,
  type AccountInfo,
  type AuthenticationResult,
  type Configuration,
} from '@azure/msal-browser'

import type { AccountingScopeFixture, AuthScopeResolution } from './auth-scope-adapter'
import type { AuthClient } from './auth-client'
import { withSessionDeadline } from './session'

export interface MsalAuthConfig {
  clientId: string
  authority: string
  apiScope: string
  redirectUri?: string
}

export class AuthStepUpRequiredError extends Error {
  constructor() {
    super('Additional verification is required to continue.')
    this.name = 'AuthStepUpRequiredError'
  }
}

export function createMsalAuthClient(config: MsalAuthConfig): AuthClient {
  const msalConfiguration: Configuration = {
    auth: {
      clientId: config.clientId,
      authority: config.authority,
      redirectUri: config.redirectUri ?? window.location.origin,
    },
    cache: {
      cacheLocation: 'sessionStorage',
    },
  }
  const application = new PublicClientApplication(msalConfiguration)
  const scopes = [config.apiScope]
  let account: AccountInfo | null = null

  const unauthenticated = (reason: string): AuthScopeResolution => ({ status: 'unauthenticated', reason })
  const resolutionFromResult = (result: AuthenticationResult): AuthScopeResolution => withSessionDeadline({
    status: 'authenticated',
    actorLabel: result.account?.username?.trim() || result.account?.name?.trim() || account?.username?.trim() || 'Authenticated application user',
    availableScopes: [] as readonly AccountingScopeFixture[],
    initialScopeId: null,
    expiresAt: result.expiresOn ?? undefined,
  })

  const acquire = async (forceRefresh = false) => {
    if (!account) throw new Error('authentication account is unavailable')
    try {
      return await application.acquireTokenSilent({ account, scopes, forceRefresh })
    } catch (error) {
      if (error instanceof InteractionRequiredAuthError) throw new AuthStepUpRequiredError()
      throw error
    }
  }

  return {
    async initialize() {
      await application.initialize()
      const redirectResult = await application.handleRedirectPromise()
      account = redirectResult?.account ?? application.getAllAccounts()[0] ?? null
      if (!account) return unauthenticated('Sign-in is required to access TALLY.')
      try {
        return resolutionFromResult(await acquire())
      } catch (error) {
        if (error instanceof AuthStepUpRequiredError) return { status: 'step-up-required', reason: error.message }
        return unauthenticated('Sign-in could not be completed. Please try again.')
      }
    },
    login: async () => {
      await application.loginRedirect({ scopes })
    },
    logout: async () => {
      await application.logoutRedirect({ account: account ?? undefined })
    },
    getAccessToken: async () => {
      const result = await acquire()
      if (!result.accessToken) throw new Error('authentication token is unavailable')
      return result.accessToken
    },
    refreshAccessToken: async () => {
      const result = await acquire(true)
      if (!result.accessToken) throw new Error('authentication token is unavailable')
      return result.accessToken
    },
  }
}
