export class AuthenticationRefreshError extends Error {
  constructor(message = 'Authentication expired; unsent work was preserved.') {
    super(message)
    this.name = 'AuthenticationRefreshError'
  }
}

export interface AuthenticatedFetchOptions {
  getAccessToken: () => Promise<string>
  refreshAccessToken: () => Promise<string>
  onRefreshFailure?: (error: unknown) => void
  fetchImplementation?: typeof fetch
}

function requestWithBearer(input: RequestInfo | URL, init: RequestInit | undefined, token: string) {
  const headers = new Headers(input instanceof Request ? input.headers : undefined)
  new Headers(init?.headers).forEach((value, key) => headers.set(key, value))
  headers.set('Authorization', `Bearer ${token}`)
  return new Request(input, { ...init, headers })
}

export function createAuthenticatedFetch(options: AuthenticatedFetchOptions): typeof fetch {
  const fetchImplementation = options.fetchImplementation ?? globalThis.fetch
  return async (input, init) => {
    const firstToken = await options.getAccessToken()
    const firstResponse = await fetchImplementation(requestWithBearer(input, init, firstToken))
    if (firstResponse.status !== 401) return firstResponse

    try {
      const refreshedToken = await options.refreshAccessToken()
      const retriedResponse = await fetchImplementation(requestWithBearer(input, init, refreshedToken))
      if (retriedResponse.status === 401) {
        const error = new Error('Authentication retry was rejected')
        options.onRefreshFailure?.(error)
        throw new AuthenticationRefreshError()
      }
      return retriedResponse
    } catch (error) {
      if (error instanceof AuthenticationRefreshError) throw error
      options.onRefreshFailure?.(error)
      throw new AuthenticationRefreshError()
    }
  }
}
