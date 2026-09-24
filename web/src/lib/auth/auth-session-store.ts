import { useMemo, useSyncExternalStore } from 'react'

import type { AuthScopeAdapter, AuthScopeResolution } from './auth-scope-adapter'
import { configureAPIAuthentication } from './api-auth'
import type { AuthClient } from './auth-client'
import { AuthStepUpRequiredError } from './msal-auth-client'

export interface AuthSessionSnapshot {
  resolution: AuthScopeResolution
  expiryWarning: boolean
}

interface AuthSessionStore {
  subscribe(listener: () => void): () => void
  getSnapshot(): AuthSessionSnapshot
  getServerSnapshot(): AuthSessionSnapshot
}

const expiredResolution = (): AuthScopeResolution => ({
  status: 'expired',
  reason: 'Your session expired. Unsent work was preserved.',
})

export function createAuthSessionStore(authScopeAdapter: AuthScopeAdapter | undefined, authClient: AuthClient): AuthSessionStore {
  let snapshot: AuthSessionSnapshot = {
    resolution: authScopeAdapter?.resolve() ?? authClient.initialResolution ?? { status: 'loading' },
    expiryWarning: false,
  }
  let started = false
  let warningTimer: ReturnType<typeof setTimeout> | undefined
  let expiryTimer: ReturnType<typeof setTimeout> | undefined
  const listeners = new Set<() => void>()

  const notify = () => listeners.forEach((listener) => listener())
  const clearTimers = () => {
    if (warningTimer) clearTimeout(warningTimer)
    if (expiryTimer) clearTimeout(expiryTimer)
    warningTimer = undefined
    expiryTimer = undefined
  }

  const applyResolution = (resolution: AuthScopeResolution) => {
    clearTimers()
    snapshot = { resolution, expiryWarning: false }
    if (resolution.status === 'authenticated' && resolution.expiresAt && resolution.warningAt) {
      const warningAt = resolution.warningAt.getTime()
      const expiresAt = resolution.expiresAt.getTime()
      warningTimer = setTimeout(() => {
        if (snapshot.resolution !== resolution) return
        snapshot = { ...snapshot, expiryWarning: true }
        notify()
      }, Math.max(0, warningAt - Date.now()))
      expiryTimer = setTimeout(() => {
        if (snapshot.resolution !== resolution) return
        applyResolution(expiredResolution())
      }, Math.max(0, expiresAt - Date.now()))
    }
    notify()
  }

  const handleRefreshFailure = (error: unknown) => {
    if (error instanceof AuthStepUpRequiredError) {
      applyResolution({ status: 'step-up-required', reason: error.message })
      return
    }
    applyResolution(expiredResolution())
  }

  const start = () => {
    if (started) return
    started = true
    configureAPIAuthentication(authClient, handleRefreshFailure)
    if (authScopeAdapter) {
      applyResolution(snapshot.resolution)
      return
    }
    void authClient.initialize().then(applyResolution).catch(() => {
      applyResolution({ status: 'unauthenticated', reason: 'Sign-in could not be completed. Please try again.' })
    })
  }

  const subscribe = (listener: () => void) => {
    listeners.add(listener)
    start()
    return () => listeners.delete(listener)
  }

  const getSnapshot = () => snapshot
  return { subscribe, getSnapshot, getServerSnapshot: getSnapshot }
}

export function useAuthSession(authScopeAdapter: AuthScopeAdapter | undefined, authClient: AuthClient): AuthSessionSnapshot {
  const store = useMemo(() => createAuthSessionStore(authScopeAdapter, authClient), [authScopeAdapter, authClient])
  return useSyncExternalStore(store.subscribe, store.getSnapshot, store.getServerSnapshot)
}
