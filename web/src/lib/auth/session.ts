import type { AuthScopeResolution } from './auth-scope-adapter'

export const GENERAL_SESSION_TIMEOUT_MS = 30 * 60 * 1000
export const PRIVILEGED_SESSION_TIMEOUT_MS = 15 * 60 * 1000
export const SESSION_WARNING_LEAD_MS = 2 * 60 * 1000

export function sessionTimeoutMs(sessionClass: 'general' | 'privileged' = 'general') {
  return sessionClass === 'privileged' ? PRIVILEGED_SESSION_TIMEOUT_MS : GENERAL_SESSION_TIMEOUT_MS
}

export function resolveSessionDeadline(
  startedAt: Date,
  sessionClass: 'general' | 'privileged' = 'general',
  tokenExpiresAt?: Date,
) {
  const inactivityDeadline = new Date(startedAt.getTime() + sessionTimeoutMs(sessionClass))
  const expiresAt = tokenExpiresAt && tokenExpiresAt < inactivityDeadline ? tokenExpiresAt : inactivityDeadline
  return {
    expiresAt,
    warningAt: new Date(expiresAt.getTime() - SESSION_WARNING_LEAD_MS),
  }
}

export function withSessionDeadline(resolution: AuthScopeResolution, startedAt = new Date()): AuthScopeResolution {
  if (resolution.status !== 'authenticated') return resolution
  const deadline = resolveSessionDeadline(startedAt, resolution.sessionClass, resolution.expiresAt)
  return { ...resolution, ...deadline }
}
