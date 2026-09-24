import { describe, expect, beforeEach, it, vi } from 'vitest'

const mocks = vi.hoisted(() => {
  class MockInteractionRequiredAuthError extends Error {}
  const account = { homeAccountId: 'home-id', localAccountId: 'local-id', username: 'fixture@example.test' }
  const application = {
    initialize: vi.fn(),
    handleRedirectPromise: vi.fn(),
    getAllAccounts: vi.fn(),
    acquireTokenSilent: vi.fn(),
    loginRedirect: vi.fn(),
    logoutRedirect: vi.fn(),
  }
  const configuration = vi.fn()
  class MockPublicClientApplication {
    constructor(config: unknown) {
      configuration(config)
      return application
    }
  }
  return { account, application, configuration, MockInteractionRequiredAuthError, MockPublicClientApplication }
})

vi.mock('@azure/msal-browser', () => ({
  InteractionRequiredAuthError: mocks.MockInteractionRequiredAuthError,
  PublicClientApplication: mocks.MockPublicClientApplication,
}))

import { AuthStepUpRequiredError, createMsalAuthClient } from './msal-auth-client'

describe('MSAL authentication client', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.application.initialize.mockResolvedValue(undefined)
    mocks.application.handleRedirectPromise.mockResolvedValue(null)
    mocks.application.getAllAccounts.mockReturnValue([mocks.account])
    mocks.application.acquireTokenSilent.mockResolvedValue({
      account: mocks.account,
      accessToken: 'memory-only-token',
      expiresOn: new Date('2026-09-23T09:00:00.000Z'),
    })
    mocks.application.loginRedirect.mockResolvedValue(undefined)
    mocks.application.logoutRedirect.mockResolvedValue(undefined)
  })

  it('initializes a session, supports login, silent acquisition, and refresh', async () => {
    const client = createMsalAuthClient({ clientId: 'client-id', authority: 'https://tenant.ciamlogin.com/tenant-id', apiScope: 'api://tally/.default' })
    const resolution = await client.initialize()

    expect(resolution).toEqual(expect.objectContaining({ status: 'authenticated', actorLabel: mocks.account.username }))
    expect(mocks.configuration).toHaveBeenCalledWith(expect.objectContaining({
      auth: expect.objectContaining({ authority: 'https://tenant.ciamlogin.com/tenant-id' }),
    }))
    await client.login()
    expect(mocks.application.loginRedirect).toHaveBeenCalledOnce()
    expect(await client.getAccessToken()).toBe('memory-only-token')
    expect(await client.refreshAccessToken()).toBe('memory-only-token')
    expect(mocks.application.acquireTokenSilent).toHaveBeenLastCalledWith(expect.objectContaining({ forceRefresh: true }))
  })

  it('surfaces step-up-required when silent acquisition requires interaction', async () => {
    mocks.application.acquireTokenSilent.mockRejectedValue(new mocks.MockInteractionRequiredAuthError())
    const client = createMsalAuthClient({ clientId: 'client-id', authority: 'https://tenant.ciamlogin.com/tenant-id', apiScope: 'api://tally/.default' })

    const resolution = await client.initialize()

    expect(resolution).toEqual({ status: 'step-up-required', reason: 'Additional verification is required to continue.' })
    await expect(client.getAccessToken()).rejects.toBeInstanceOf(AuthStepUpRequiredError)
  })
})
