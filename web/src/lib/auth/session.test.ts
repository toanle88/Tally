import { describe, expect, it } from 'vitest'

import { GENERAL_SESSION_TIMEOUT_MS, PRIVILEGED_SESSION_TIMEOUT_MS, SESSION_WARNING_LEAD_MS, resolveSessionDeadline } from './session'

describe('session deadline policy', () => {
  const startedAt = new Date('2026-09-23T08:30:00.000Z')

  it('warns two minutes before the general 30-minute timeout', () => {
    const deadline = resolveSessionDeadline(startedAt, 'general')
    expect(deadline.expiresAt.getTime() - startedAt.getTime()).toBe(GENERAL_SESSION_TIMEOUT_MS)
    expect(deadline.expiresAt.getTime() - deadline.warningAt.getTime()).toBe(SESSION_WARNING_LEAD_MS)
  })

  it('uses the shorter privileged timeout and never extends an earlier token expiry', () => {
    const tokenExpiry = new Date(startedAt.getTime() + 5 * 60 * 1000)
    const deadline = resolveSessionDeadline(startedAt, 'privileged', tokenExpiry)
    expect(deadline.expiresAt).toEqual(tokenExpiry)
    expect(deadline.expiresAt.getTime() - startedAt.getTime()).toBeLessThan(PRIVILEGED_SESSION_TIMEOUT_MS)
  })
})
