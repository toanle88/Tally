import { describe, expect, it, vi } from 'vitest'

import { AuthenticationRefreshError, createAuthenticatedFetch } from './authenticated-fetch'

describe('createAuthenticatedFetch', () => {
  it('sends the current token without refreshing successful requests', async () => {
    const calls: Request[] = []
    const fetchImplementation: typeof fetch = vi.fn(async (input, init) => {
      calls.push(new Request(input, init))
      return new Response(null, { status: 204 })
    })
    const fetchWithAuth = createAuthenticatedFetch({
      getAccessToken: async () => 'first-token',
      refreshAccessToken: async () => 'unused-token',
      fetchImplementation,
    })

    const response = await fetchWithAuth('https://tally.test/api/v1/future-operation', { method: 'POST' })

    expect(response.status).toBe(204)
    expect(calls).toHaveLength(1)
    expect(calls[0].headers.get('Authorization')).toBe('Bearer first-token')
  })

  it('refreshes once and retries a 401 without mutating the original request', async () => {
    const calls: Request[] = []
    const responses = [new Response(null, { status: 401 }), new Response(null, { status: 204 })]
    const fetchImplementation: typeof fetch = vi.fn(async (input, init) => {
      calls.push(new Request(input, init))
      return responses.shift() as Response
    })
    const refreshAccessToken = vi.fn(async () => 'refreshed-token')
    const fetchWithAuth = createAuthenticatedFetch({
      getAccessToken: async () => 'expired-token',
      refreshAccessToken,
      fetchImplementation,
    })

    const response = await fetchWithAuth('https://tally.test/api/v1/future-operation', { method: 'POST', body: '{}' })

    expect(response.status).toBe(204)
    expect(refreshAccessToken).toHaveBeenCalledOnce()
    expect(calls).toHaveLength(2)
    expect(calls[0].headers.get('Authorization')).toBe('Bearer expired-token')
    expect(calls[1].headers.get('Authorization')).toBe('Bearer refreshed-token')
  })

  it('fails safely after one refresh failure', async () => {
    const onRefreshFailure = vi.fn()
    const fetchImplementation: typeof fetch = vi.fn(async () => new Response(null, { status: 401 }))
    const fetchWithAuth = createAuthenticatedFetch({
      getAccessToken: async () => 'expired-token',
      refreshAccessToken: async () => { throw new Error('interaction required') },
      onRefreshFailure,
      fetchImplementation,
    })

    await expect(fetchWithAuth('https://tally.test/api/v1/future-operation')).rejects.toBeInstanceOf(AuthenticationRefreshError)
    expect(onRefreshFailure).toHaveBeenCalledOnce()
    expect(fetchImplementation).toHaveBeenCalledOnce()
  })

  it('requires reauthentication when the single retry is still unauthorized', async () => {
    const onRefreshFailure = vi.fn()
    const fetchImplementation: typeof fetch = vi.fn(async () => new Response(null, { status: 401 }))
    const fetchWithAuth = createAuthenticatedFetch({
      getAccessToken: async () => 'expired-token',
      refreshAccessToken: async () => 'refreshed-token',
      onRefreshFailure,
      fetchImplementation,
    })

    await expect(fetchWithAuth('https://tally.test/api/v1/future-operation')).rejects.toBeInstanceOf(AuthenticationRefreshError)
    expect(onRefreshFailure).toHaveBeenCalledOnce()
    expect(fetchImplementation).toHaveBeenCalledTimes(2)
  })
})
