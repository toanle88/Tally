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
      ['IAM-WS-01', '/administration/identity-access'],
      ['IAM-SCR-04', '/identity-access/iam-scr-04'],
      ['OMD-WS-01', '/master-data/omd-ws-01'],
      ['OMD-SCR-01', '/master-data/omd-scr-01'],
      ['OMD-SCR-02', '/master-data/omd-scr-02'],
      ['OMD-SCR-03', '/master-data/omd-scr-03'],
      ['OMD-SCR-04', '/master-data/omd-scr-04'],
      ['OMD-SCR-05', '/master-data/omd-scr-05'],
      ['COA-WS-01', '/coa-segments/coa-ws-01'],
      ['COA-SCR-01', '/coa-segments/coa-scr-01'],
      ['COA-SCR-02', '/coa-segments/coa-scr-02'],
      ['COA-SCR-03', '/coa-segments/coa-scr-03'],
      ['COA-SCR-04', '/coa-segments/coa-scr-04'],
      ['GL-SCR-04', '/general-ledger/gl-scr-04'],
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
