import { createBrowserRouter, RouterProvider, redirect } from 'react-router-dom'
import { Layout } from './components/layouts/Layout'
import { Dashboard } from './pages/Dashboard'
import { Storage } from './pages/Storage'
import SharesIndex from './routes/shares/index'
import { Apps } from './pages/Apps'
import { Settings } from './pages/Settings'
import { Remote } from './pages/Remote'
import { Login } from './pages/Login'
import { PoolsCreate } from './pages/PoolsCreate'
import { PoolDetails } from './pages/PoolDetails'
import { SettingsUpdates } from './pages/SettingsUpdates'
import Setup from './pages/Setup'

const router = createBrowserRouter([
	{
		path: '/',
		loader: async () => {
			// On init: check setup, then auth
			try {
				const stRes = await fetch('/api/setup/state', { credentials: 'include' })
				if (stRes.status === 200) {
					const st = await stRes.json()
					if (st?.firstBoot) return redirect('/setup')
				} else if (stRes.status === 410) {
					// setup completed; continue to auth check
				}
			} catch {}
			// Auth check
			try {
				const me = await fetch('/api/auth/me', { credentials: 'include' })
				if (me.ok) return null
			} catch {}
			return redirect('/login')
		},
		element: <Layout />,
		children: [
			{ index: true, element: <Dashboard /> },
			{ path: 'storage', element: <Storage /> },
			{ path: 'shares', element: <SharesIndex /> },
			{ path: 'apps', element: <Apps /> },
			{ path: 'settings', element: <Settings /> },
			{ path: 'settings/updates', element: <SettingsUpdates /> },
			{ path: 'remote', element: <Remote /> },
			{ path: 'storage/create', element: <PoolsCreate /> },
			{ path: 'storage/:id', element: <PoolDetails /> },
		],
	},
	{ path: '/login', element: <Login /> },
	{ path: '/setup', element: <Setup /> },
])

export default function App() {
	return <RouterProvider router={router} />
}


