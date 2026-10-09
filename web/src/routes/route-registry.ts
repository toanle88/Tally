export type NavigationArea =
  | 'Home'
  | 'Work'
  | 'Records'
  | 'Approvals'
  | 'Exceptions'
  | 'Reports'
  | 'Administration'
  | 'Audit'

export type ScreenId = 'XCT-WS-01' | 'XCT-SCR-01' | 'CON-SCR-01' | 'IAM-WS-01' | 'IAM-SCR-01' | 'IAM-SCR-04' | 'OMD-WS-01' | 'OMD-SCR-01' | 'OMD-SCR-02' | 'OMD-SCR-03' | 'OMD-SCR-04' | 'OMD-SCR-05' | 'COA-WS-01' | 'COA-SCR-01' | 'COA-SCR-02' | 'COA-SCR-03' | 'COA-SCR-04'

export type RouteId =
  | 'home'
  | 'work'
  | 'records'
  | 'approvals'
  | 'exceptions'
  | 'reports'
  | 'administration'
  | 'iamWorklist'
  | 'iamEmergencyAccess'
  | 'omdWorklist'
  | 'omdDetail'
  | 'omdPartyDetail'
  | 'omdCustomerProfileDetail'
  | 'omdFiscalCalendarDetail'
  | 'omdPublicationReview'
  | 'coaWorklist'
  | 'coaDetail'
  | 'coaValue'
  | 'coaCombinationValidator'
  | 'coaChangeRequest'
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
  { id: 'iamWorklist', path: '/administration/identity-access', title: 'Users and access assignments', kind: 'operational', navigationArea: 'Administration', screenId: 'IAM-WS-01', requiresScope: true },
  { id: 'iamEmergencyAccess', path: '/identity-access/iam-scr-04', title: 'Emergency access grant and review', kind: 'operational', navigationArea: 'Administration', screenId: 'IAM-SCR-04', requiresScope: true },
  { id: 'omdWorklist', path: '/master-data/omd-ws-01', title: 'Legal-entity master-data worklist', kind: 'operational', navigationArea: 'Records', screenId: 'OMD-WS-01', requiresScope: true },
  { id: 'omdDetail', path: '/master-data/omd-scr-01', title: 'Legal-entity record', kind: 'operational', navigationArea: 'Records', screenId: 'OMD-SCR-01', requiresScope: true },
  { id: 'omdPartyDetail', path: '/master-data/omd-scr-02', title: 'Party record', kind: 'operational', navigationArea: 'Records', screenId: 'OMD-SCR-02', requiresScope: true },
  { id: 'omdCustomerProfileDetail', path: '/master-data/omd-scr-03', title: 'Customer-profile record', kind: 'operational', navigationArea: 'Records', screenId: 'OMD-SCR-03', requiresScope: true },
  { id: 'omdFiscalCalendarDetail', path: '/master-data/omd-scr-04', title: 'Fiscal-calendar editor', kind: 'operational', navigationArea: 'Records', screenId: 'OMD-SCR-04', requiresScope: true },
  { id: 'omdPublicationReview', path: '/master-data/omd-scr-05', title: 'Approved master-data publication review', kind: 'operational', navigationArea: 'Records', screenId: 'OMD-SCR-05', requiresScope: true },
  { id: 'coaWorklist', path: '/coa-segments/coa-ws-01', title: 'COA segment administration worklist', kind: 'operational', navigationArea: 'Records', screenId: 'COA-WS-01', requiresScope: true },
  { id: 'coaDetail', path: '/coa-segments/coa-scr-01', title: 'COA segment definition', kind: 'operational', navigationArea: 'Records', screenId: 'COA-SCR-01', requiresScope: true },
  { id: 'coaValue', path: '/coa-segments/coa-scr-02', title: 'COA segment value', kind: 'operational', navigationArea: 'Records', screenId: 'COA-SCR-02', requiresScope: true },
  { id: 'coaCombinationValidator', path: '/coa-segments/coa-scr-03', title: 'COA segment combination validator', kind: 'operational', navigationArea: 'Records', screenId: 'COA-SCR-03', requiresScope: true },
  { id: 'coaChangeRequest', path: '/coa-segments/coa-scr-04', title: 'COA segment change request', kind: 'operational', navigationArea: 'Records', screenId: 'COA-SCR-04', requiresScope: true },
  { id: 'audit', path: '/audit', title: 'Audit', kind: 'area', navigationArea: 'Audit', requiresScope: true },
  { id: 'xctWorklist', path: '/operations/xct-ws-01', title: 'Cross-context event exception worklist', kind: 'operational', navigationArea: 'Exceptions', screenId: 'XCT-WS-01', requiresScope: true },
  { id: 'xctDetail', path: '/operations/xct-scr-01', title: 'Cross-context event outcome detail', kind: 'operational', navigationArea: 'Exceptions', screenId: 'XCT-SCR-01', requiresScope: true },
  { id: 'conConflict', path: '/operations/con-scr-01', title: 'Concurrency conflict and safe-retry view', kind: 'operational', navigationArea: 'Exceptions', screenId: 'CON-SCR-01', requiresScope: true },
  { id: 'developmentExamples', path: '/development/examples', title: 'Development examples', kind: 'development', requiresScope: false },
] as const satisfies readonly RouteDefinition[]

export const navigationItems = routeRegistry.filter(
  (route): route is Extract<(typeof routeRegistry)[number], { kind: 'area' }> => route.kind === 'area',
)
