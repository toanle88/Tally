export type NavigationArea =
  | 'Home'
  | 'Work'
  | 'Records'
  | 'Approvals'
  | 'Exceptions'
  | 'Reports'
  | 'Administration'
  | 'Audit'

export type ScreenId = 'XCT-WS-01' | 'XCT-SCR-01' | 'CON-SCR-01'

export type RouteId =
  | 'home'
  | 'work'
  | 'records'
  | 'approvals'
  | 'exceptions'
  | 'reports'
  | 'administration'
  | 'audit'
  | 'xctWorklist'
  | 'xctDetail'
  | 'conConflict'
  | 'developmentExamples'

export interface RouteDefinition {
  id: RouteId
  path: string
  title: string
  kind: 'area' | 'operational' | 'development'
  navigationArea?: NavigationArea
  screenId?: ScreenId
  requiresScope: boolean
}

export const routeRegistry = [
  { id: 'home', path: '/', title: 'Home', kind: 'area', navigationArea: 'Home', requiresScope: false },
  { id: 'work', path: '/work', title: 'Work', kind: 'area', navigationArea: 'Work', requiresScope: true },
  { id: 'records', path: '/records', title: 'Records', kind: 'area', navigationArea: 'Records', requiresScope: true },
  { id: 'approvals', path: '/approvals', title: 'Approvals', kind: 'area', navigationArea: 'Approvals', requiresScope: true },
  { id: 'exceptions', path: '/exceptions', title: 'Exceptions', kind: 'area', navigationArea: 'Exceptions', requiresScope: true },
  { id: 'reports', path: '/reports', title: 'Reports', kind: 'area', navigationArea: 'Reports', requiresScope: true },
  { id: 'administration', path: '/administration', title: 'Administration', kind: 'area', navigationArea: 'Administration', requiresScope: true },
  { id: 'audit', path: '/audit', title: 'Audit', kind: 'area', navigationArea: 'Audit', requiresScope: true },
  { id: 'xctWorklist', path: '/operations/xct-ws-01', title: 'Cross-context event exception worklist', kind: 'operational', navigationArea: 'Exceptions', screenId: 'XCT-WS-01', requiresScope: true },
  { id: 'xctDetail', path: '/operations/xct-scr-01', title: 'Cross-context event outcome detail', kind: 'operational', navigationArea: 'Exceptions', screenId: 'XCT-SCR-01', requiresScope: true },
  { id: 'conConflict', path: '/operations/con-scr-01', title: 'Concurrency conflict and safe-retry view', kind: 'operational', navigationArea: 'Exceptions', screenId: 'CON-SCR-01', requiresScope: true },
  { id: 'developmentExamples', path: '/development/examples', title: 'Development examples', kind: 'development', requiresScope: false },
] as const satisfies readonly RouteDefinition[]

export const navigationItems = routeRegistry.filter(
  (route): route is Extract<(typeof routeRegistry)[number], { kind: 'area' }> => route.kind === 'area',
)
