import { describe, expect, it } from 'vitest'

import { navigationItems, routeRegistry } from './route-registry'
import { createAppRoutes, navigationAreaForPath } from './router'

describe('route registry', () => {
  it('defines stable global navigation paths', () => {
    expect(navigationItems.map((route) => [route.title, route.path])).toEqual([
      ['Home', '/'],
      ['Work', '/work'],
      ['Records', '/records'],
      ['Approvals', '/approvals'],
      ['Exceptions', '/exceptions'],
      ['Reports', '/reports'],
      ['Administration', '/administration'],
      ['Audit', '/audit'],
    ])
  })

  it('keeps operational contracts discoverable without implementing capabilities', () => {
    expect(routeRegistry.filter((route) => route.kind === 'operational').map((route) => [route.screenId, route.path])).toEqual([
      ['XCT-WS-01', '/operations/xct-ws-01'],
      ['XCT-SCR-01', '/operations/xct-scr-01'],
      ['CON-SCR-01', '/operations/con-scr-01'],
    ])
  })

  it('maps only registered navigation paths to an area', () => {
    expect(navigationAreaForPath('/exceptions')).toBe('Exceptions')
    expect(navigationAreaForPath('/operations/xct-ws-01')).toBe('Exceptions')
    expect(navigationAreaForPath('/not-a-route')).toBeUndefined()
  })

  it('passes stable registry IDs into the router route tree', () => {
    const childRoutes = createAppRoutes()[0].children ?? []
    expect(childRoutes.slice(0, routeRegistry.length).map((route) => route.id)).toEqual(routeRegistry.map((route) => route.id))
  })
})
