import { createContext, useContext, type ReactNode } from 'react'
import { createBrowserRouter, NavLink, Outlet, type RouteObject } from 'react-router-dom'

import { AppShell } from '@/app/app-shell'
import { DevelopmentExamples } from '@/app/development-examples'
import { AccountingScopeSelector } from '@/components/accounting-scope-selector'
import { StatusBadge } from '@/components/ui'
import type { AuthScopeResolution } from '@/lib/auth/auth-scope-adapter'
import { useScopeContext } from '@/lib/scope/scope-context'

import { navigationItems, routeRegistry, type NavigationArea, type RouteDefinition } from './route-registry'

const AuthScopeContext = createContext<AuthScopeResolution | null>(null)

export function AuthScopeProvider({ resolution, children }: { resolution: AuthScopeResolution; children: ReactNode }) {
  return <AuthScopeContext.Provider value={resolution}>{children}</AuthScopeContext.Provider>
}

function useAuthScope() {
  const context = useContext(AuthScopeContext)
  if (!context) throw new Error('useAuthScope must be used within AuthScopeProvider')
  return context
}

function ShellLayout() {
  const { statusMessage } = useScopeContext()
  return <AppShell navigation={<GlobalNavigation />} scopeContext={<AccountingScopeSelector />} statusFeedback={<StatusBadge state="info" label={statusMessage} announce />}><Outlet /></AppShell>
}

function GlobalNavigation() {
  return navigationItems.map((item) => (
    <NavLink key={item.id} to={item.path} end={item.path === '/'} className={({ isActive }) => `block rounded-btn px-3 py-2 text-sm font-medium focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary ${isActive ? 'bg-primary text-primary-content' : 'hover:bg-base-200'}`}>
      {item.title}
    </NavLink>
  ))
}

function ProtectedRoute({ route, children }: { route: RouteDefinition; children: ReactNode }) {
  const resolution = useAuthScope()
  const { currentScope } = useScopeContext()
  if (resolution.status === 'unauthenticated') return <AccessState title="Authentication required" detail={resolution.reason} />
  if (route.requiresScope && !currentScope) return <AccessState title="Accounting scope required" detail="Select an available accounting scope before reviewing or initiating work." />
  return <>{children}</>
}

function AccessState({ title, detail }: { title: string; detail: string }) {
  return <section className="rounded-box border border-warning/50 bg-base-100 p-6"><h2 className="text-xl font-semibold">{title}</h2><p className="mt-2 text-base-content/75">{detail}</p></section>
}

function AreaPage({ route }: { route: RouteDefinition }) {
  return <section><h2 className="text-2xl font-semibold">{route.title}</h2><p className="mt-2 max-w-3xl text-base-content/75">This shared route is ready for a later capability delivery. No finance capability or authoritative record is connected.</p>{route.id === 'exceptions' ? <div className="mt-5 space-y-2">{routeRegistry.filter((candidate) => candidate.kind === 'operational').map((candidate) => <p key={candidate.id}><NavLink className="link link-primary" to={candidate.path}>{candidate.screenId}: {candidate.title}</NavLink></p>)}</div> : null}</section>
}

function OperationalPage({ route }: { route: RouteDefinition }) {
  return <section><p className="text-sm font-semibold uppercase tracking-wide text-primary">{route.screenId}</p><h2 className="mt-1 text-2xl font-semibold">{route.title}</h2><p className="mt-2 max-w-3xl text-base-content/75">Shared operational route contract only. Resolution remains owned by the receiving capability and is not implemented in this story.</p></section>
}

function NotFoundPage() {
  return <section><h2 className="text-2xl font-semibold">Page not found</h2><p className="mt-2 text-base-content/75">The requested route is not part of the TALLY foundation route tree.</p></section>
}

function routeElement(route: RouteDefinition) {
  const content = route.kind === 'operational' ? <OperationalPage route={route} /> : route.kind === 'development' ? <DevelopmentExamples /> : <AreaPage route={route} />
  return <ProtectedRoute route={route}>{content}</ProtectedRoute>
}

export function createAppRoutes(): RouteObject[] {
  return [{
    element: <ShellLayout />,
    children: [
      ...routeRegistry.map((route) => ({
        id: route.id,
        path: route.path,
        handle: {
          navigationArea: 'navigationArea' in route ? route.navigationArea : undefined,
          screenId: 'screenId' in route ? route.screenId : undefined,
        },
        element: routeElement(route),
      })),
      { id: 'notFound', path: '*', element: <NotFoundPage /> },
    ],
  }]
}

export function createAppRouter() {
  return createBrowserRouter(createAppRoutes())
}

export function navigationAreaForPath(path: string): NavigationArea | undefined {
  const route = routeRegistry.find((candidate) => candidate.path === path)
  return route && 'navigationArea' in route ? route.navigationArea : undefined
}
