import { createBrowserRouter, RouterProvider, redirect, Navigate } from 'react-router-dom'
import { AppShell } from './components/layout/AppShell'
import { Dashboard } from './pages/Dashboard'
import { Storage } from './pages/Storage'
import SharesIndex from './routes/shares/index'
import { Apps } from './pages/Apps'
import { Settings } from './pages/Settings'
import SettingsSchedules from './routes/settings/schedules'
import { Remote } from './pages/Remote'
import { Login } from './pages/Login'
import { PoolsCreate } from './pages/PoolsCreate'
import { PoolDetails } from './pages/PoolDetails'
import { SettingsUpdates } from './pages/SettingsUpdates'
import Setup from './pages/Setup'
import { ErrProxyMisconfigured, fetchJSON } from './api'
import { GlobalNoticeProvider, useGlobalNotice } from './lib/globalNotice'
import Banner from './components/Banner'
import HelpProxy from './pages/HelpProxy'

const router = createBrowserRouter([
	{
		path: '/',
		loader: async () => {
			// On init: check setup, then auth
			try {
				const st = await fetchJSON<any>('/api/setup/state')
				if (st?.firstBoot) return redirect('/setup')
			} catch (e: any) {
				if (e instanceof ErrProxyMisconfigured) {
					// Do not navigate; keep current page. Let global notice handle UI
					return null
				}
				// If 410: setup completed; continue to auth check
				if (e && e.status === 410) {
					// setup completed; continue to auth check
				} else {
					// ignore other errors here
				}
			}
			// Auth check
			try {
				await fetchJSON<any>('/api/auth/me')
				return null
			} catch (e: any) {
				if (e instanceof ErrProxyMisconfigured) {
					return null
				}
				return redirect('/login')
			}
		},
		element: <AppShell />,
		children: [
			{ index: true, element: <Dashboard /> },
			{ path: 'storage', element: <Storage /> },
			{ path: 'shares', element: <SharesIndex /> },
			{ path: 'apps', element: <Apps /> },
			{ path: 'settings', element: <Settings /> },
			{ path: 'settings/schedules', element: <SettingsSchedules /> },
			{ path: 'settings/updates', element: <SettingsUpdates /> },
			{ path: 'remote', element: <Remote /> },
			{ path: 'storage/create', element: <PoolsCreate /> },
			{ path: 'storage/:id', element: <PoolDetails /> },
			// Redirect old schedules route to new location
			{ path: 'schedules', element: <Navigate to="/settings/schedules" replace /> },
		],
	},
	{ path: '/login', element: <Login /> },
	{ path: '/setup', element: <Setup /> },
	{ path: '/help/proxy', element: <HelpProxy /> },
])

function AppShell() {
	const { notice } = useGlobalNotice()
	return (
		<div className="min-h-screen">
			{notice && (
				<Banner
					variant={notice.kind}
					title={notice.title}
					message={notice.message}
					action={notice.action}
				/>
			)}
			<RouterProvider router={router} />
		</div>
	)
}

export default function App() {
	return (
		<GlobalNoticeProvider>
			<AppShell />
		</GlobalNoticeProvider>
	)
}


