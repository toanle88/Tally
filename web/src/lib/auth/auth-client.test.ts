import { describe, expect, it } from 'vitest'

import { createFixtureAuthClient, createUnavailableAuthClient } from './auth-client'

describe('frontend auth mode selection', () => {
  it('requires explicit fixture construction for local fixture sessions', async () => {
    const fixture = createFixtureAuthClient({ actorLabel: 'Fixture actor', availableScopes: [], initialScopeId: null })
    const unavailable = createUnavailableAuthClient()

    expect((await fixture.initialize()).status).toBe('authenticated')
    expect((await unavailable.initialize()).status).toBe('unauthenticated')
  })
})
