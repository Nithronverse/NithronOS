import { createBrowserRouter, RouterProvider, Navigate, Outlet } from 'react-router-dom'
import { AppShell } from './components/layout/AppShell'
import { Dashboard } from './pages/Dashboard'
import { Storage } from './pages/Storage'
import { SharesList } from './pages/SharesList'
import { ShareDetails } from './pages/ShareDetails'
import { AppCatalog } from './pages/AppCatalog'
import { AppInstallWizard } from './pages/AppInstallWizard'
import { AppDetails } from './pages/AppDetails'
import { Settings } from './pages/Settings'
import { Users } from './pages/Users'
import SettingsSchedules from './routes/settings/schedules'
import { Remote } from './pages/Remote'
import { Login } from './pages/Login'
import { NetworkSettings } from './pages/NetworkSettings'
import { RemoteAccessWizard } from './pages/RemoteAccessWizard'
import { TwoFactorSettings } from './pages/TwoFactorSettings'
import { PoolsCreate } from './pages/PoolsCreate'
import { PoolDetails } from './pages/PoolDetails'
import Updates from './pages/Updates'
import Setup from './pages/Setup'
import { GlobalNoticeProvider, useGlobalNotice } from './lib/globalNotice'
import Banner from './components/Banner'
import HelpProxy from './pages/HelpProxy'
import { NotFound } from './pages/NotFound'
import { Toasts } from '@/components/ui/toast'
import { AuthProvider, AuthGuard } from './lib/auth'
import { useEffect, useState } from 'react'
import { api, APIError, ProxyMisconfiguredError } from './lib/api-client'

// ============================================================================
// Root Layout with Auth Provider
// ============================================================================

function RootLayout() {
  const { notice } = useGlobalNotice()
  
  return (
    <AuthProvider>
      <div className="min-h-screen">
        {notice && (
          <Banner
            variant={notice.kind}
            title={notice.title}
            message={notice.message}
            action={notice.action}
          />
        )}
        <Outlet />
      </div>
    </AuthProvider>
  )
}

// ============================================================================
// Protected Layout Component
// ============================================================================

function ProtectedLayout() {
  return (
    <AuthGuard requireAuth={true}>
      <AppShell>
        <Outlet />
      </AppShell>
    </AuthGuard>
  )
}

// ============================================================================
// Setup Guard Component
// ============================================================================

function SetupGuard({ children }: { children: React.ReactNode }) {
  const [loading, setLoading] = useState(true)
  const [needsSetup, setNeedsSetup] = useState(false)
  
  useEffect(() => {
    checkSetupState()
  }, [])
  
  const checkSetupState = async () => {
    try {
      const state = await api.setup.getState()
      setNeedsSetup(state.firstBoot)
    } catch (err) {
      if (err instanceof APIError && err.status === 410) {
        // Setup complete
        setNeedsSetup(false)
      } else if (err instanceof ProxyMisconfiguredError) {
        // Let global notice handle this
        setNeedsSetup(false)
      } else {
        console.error('Setup state check failed:', err)
        setNeedsSetup(false)
      }
    } finally {
      setLoading(false)
    }
  }
  
  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-muted-foreground">Loading...</div>
      </div>
    )
  }
  
  if (needsSetup) {
    return <Navigate to="/setup" replace />
  }
  
  return <>{children}</>
}

// ============================================================================
// Router Configuration
// ============================================================================

const router = createBrowserRouter([
  {
    path: '/',
    element: <RootLayout />,
    children: [
      {
        path: '/',
        element: <ProtectedLayout />,
        children: [
          { index: true, element: <Dashboard /> },
          { path: 'storage', element: <Storage /> },
          { path: 'shares', element: <SharesList /> },
          { path: 'shares/new', element: <ShareDetails /> },
          { path: 'shares/:name', element: <ShareDetails /> },
          { path: 'apps', element: <AppCatalog /> },
          { path: 'apps/install/:id', element: <AppInstallWizard /> },
          { path: 'apps/:id', element: <AppDetails /> },
          { path: 'settings', element: <Settings /> },
          { path: 'settings/users', element: <Users /> },
          { path: 'settings/schedules', element: <SettingsSchedules /> },
          { path: 'settings/updates', element: <Updates /> },
          { path: 'settings/network', element: <NetworkSettings /> },
          { path: 'settings/network/wizard', element: <RemoteAccessWizard /> },
          { path: 'settings/2fa', element: <TwoFactorSettings /> },
          { path: 'remote', element: <Remote /> },
          { path: 'storage/create', element: <PoolsCreate /> },
          { path: 'storage/:id', element: <PoolDetails /> },
          // Redirect old schedules route to new location
          { path: 'schedules', element: <Navigate to="/settings/schedules" replace /> },
        ],
      },
      {
        path: '/login',
        element: (
          <SetupGuard>
            <Login />
          </SetupGuard>
        ),
      },
      {
        path: '/setup',
        element: <Setup />,
      },
      {
        path: '/help/proxy',
        element: <HelpProxy />,
      },
      {
        path: '*',
        element: <NotFound />,
      },
    ],
  },
])

// ============================================================================
// Main App Component
// ============================================================================

export default function App() {
  return (
    <GlobalNoticeProvider>
      <Toasts />
      <RouterProvider router={router} />
    </GlobalNoticeProvider>
  )
}