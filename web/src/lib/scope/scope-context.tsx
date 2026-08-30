import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from 'react'

import type { AccountingScopeFixture } from '@/lib/auth/auth-scope-adapter'

export interface ScopeSnapshot {
  scopeId: string
  revision: number
}

export interface ScopeChangeEvent {
  previous: ScopeSnapshot | null
  next: ScopeSnapshot
}

export interface ScopeChangeEffects {
  cancelInFlightWork: () => void
  clearScopeBoundQueryState: () => void
  onScopeChange?: (event: ScopeChangeEvent) => void
}

export interface ScopeContextValue {
  currentScope: AccountingScopeFixture | null
  availableScopes: readonly AccountingScopeFixture[]
  scopeRevision: number
  hasUnsavedChanges: boolean
  pendingScope: AccountingScopeFixture | null
  setHasUnsavedChanges: (value: boolean) => void
  requestScopeChange: (scopeId: string) => void
  confirmScopeChange: () => void
  cancelScopeChange: () => void
  captureScopeSnapshot: () => ScopeSnapshot | null
  isCurrentScope: (snapshot: ScopeSnapshot | null) => boolean
  runIfCurrentScope: (snapshot: ScopeSnapshot | null, submit: () => void) => boolean
  lastChangeError: string | null
  statusMessage: string
}

export interface ScopeProviderProps {
  availableScopes: readonly AccountingScopeFixture[]
  initialScopeId: string | null
  effects: ScopeChangeEffects
  children: ReactNode
}

const ScopeContext = createContext<ScopeContextValue | null>(null)

export function ScopeProvider({
  availableScopes,
  initialScopeId,
  effects,
  children,
}: ScopeProviderProps) {
  const initialScope = availableScopes.find((scope) => scope.id === initialScopeId) ?? null
  const [currentScopeId, setCurrentScopeId] = useState<string | null>(initialScope?.id ?? null)
  const [scopeRevision, setScopeRevision] = useState(0)
  const [hasUnsavedChanges, setHasUnsavedChanges] = useState(false)
  const [pendingScopeId, setPendingScopeId] = useState<string | null>(null)
  const [lastChangeError, setLastChangeError] = useState<string | null>(null)
  const [statusMessage, setStatusMessage] = useState('Foundation preview only')

  const currentScope = availableScopes.find((scope) => scope.id === currentScopeId) ?? null
  const pendingScope = availableScopes.find((scope) => scope.id === pendingScopeId) ?? null

  const applyScopeChange = useCallback((nextScope: AccountingScopeFixture) => {
    if (nextScope.id === currentScopeId) {
      setPendingScopeId(null)
      return
    }

    const previousSnapshot = currentScope
      ? { scopeId: currentScope.id, revision: scopeRevision }
      : null
    const nextSnapshot = { scopeId: nextScope.id, revision: scopeRevision + 1 }

    try {
      effects.cancelInFlightWork()
      effects.clearScopeBoundQueryState()
      effects.onScopeChange?.({ previous: previousSnapshot, next: nextSnapshot })
    } catch {
      const message = 'The accounting scope could not be changed safely. Current work was preserved.'
      setLastChangeError(message)
      setStatusMessage(message)
      return
    }

    setLastChangeError(null)
    setCurrentScopeId(nextScope.id)
    setScopeRevision(nextSnapshot.revision)
    setPendingScopeId(null)
    setHasUnsavedChanges(false)
    setStatusMessage(`Accounting scope refreshed for ${nextScope.legalEntity.name}.`)
  }, [currentScope, currentScopeId, effects, scopeRevision])

  const requestScopeChange = useCallback((scopeId: string) => {
    const nextScope = availableScopes.find((scope) => scope.id === scopeId)
    if (!nextScope) {
      return
    }

    if (hasUnsavedChanges) {
      setPendingScopeId(nextScope.id)
      return
    }

    applyScopeChange(nextScope)
  }, [applyScopeChange, availableScopes, hasUnsavedChanges])

  const confirmScopeChange = useCallback(() => {
    if (pendingScope) {
      applyScopeChange(pendingScope)
    }
  }, [applyScopeChange, pendingScope])

  const cancelScopeChange = useCallback(() => setPendingScopeId(null), [])
  const captureScopeSnapshot = useCallback(
    () => (currentScope ? { scopeId: currentScope.id, revision: scopeRevision } : null),
    [currentScope, scopeRevision],
  )
  const isCurrentScope = useCallback(
    (snapshot: ScopeSnapshot | null) =>
      snapshot !== null &&
      currentScope !== null &&
      snapshot.scopeId === currentScope.id &&
      snapshot.revision === scopeRevision,
    [currentScope, scopeRevision],
  )
  const runIfCurrentScope = useCallback(
    (snapshot: ScopeSnapshot | null, submit: () => void) => {
      if (isCurrentScope(snapshot)) {
        submit()
        return true
      }

      setStatusMessage('This work belongs to an older accounting scope. Review the refreshed scope before submitting.')
      return false
    },
    [isCurrentScope],
  )

  const value = useMemo<ScopeContextValue>(
    () => ({
      currentScope,
      availableScopes,
      scopeRevision,
      hasUnsavedChanges,
      pendingScope,
      setHasUnsavedChanges,
      requestScopeChange,
      confirmScopeChange,
      cancelScopeChange,
      captureScopeSnapshot,
      isCurrentScope,
      runIfCurrentScope,
      lastChangeError,
      statusMessage,
    }),
    [availableScopes, cancelScopeChange, captureScopeSnapshot, confirmScopeChange, currentScope, hasUnsavedChanges, isCurrentScope, lastChangeError, pendingScope, requestScopeChange, runIfCurrentScope, scopeRevision, statusMessage],
  )

  return <ScopeContext.Provider value={value}>{children}</ScopeContext.Provider>
}

export function useScopeContext() {
  const context = useContext(ScopeContext)
  if (!context) {
    throw new Error('useScopeContext must be used within ScopeProvider')
  }
  return context
}
