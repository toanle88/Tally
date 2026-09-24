import { client } from '@/generated/api/client.gen'

import type { AuthClient } from './auth-client'
import { createAuthenticatedFetch } from './authenticated-fetch'

export function configureAPIAuthentication(authClient: AuthClient, onRefreshFailure: (error: unknown) => void) {
  client.setConfig({
    auth: () => authClient.getAccessToken(),
    fetch: createAuthenticatedFetch({
      getAccessToken: () => authClient.getAccessToken(),
      refreshAccessToken: () => authClient.refreshAccessToken(),
      onRefreshFailure,
    }),
  })
}
