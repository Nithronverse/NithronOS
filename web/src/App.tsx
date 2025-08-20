import { createBrowserRouter, RouterProvider, redirect } from 'react-router-dom'
import { Layout } from './components/layouts/Layout'
import { Dashboard } from './pages/Dashboard'
import { Storage } from './pages/Storage'
import { Shares } from './pages/Shares'
import { Apps } from './pages/Apps'
import { Settings } from './pages/Settings'
import { Remote } from './pages/Remote'
import { Login } from './pages/Login'

const router = createBrowserRouter([
	{
		path: '/',
		loader: async () => {
			// simple guard: require session cookie
			const hasSess = document.cookie.includes('nos_sess=')
			if (!hasSess) throw redirect('/login')
			return null
		},
		element: <Layout />,
		children: [
			{ index: true, element: <Dashboard /> },
			{ path: 'storage', element: <Storage /> },
			{ path: 'shares', element: <Shares /> },
			{ path: 'apps', element: <Apps /> },
			{ path: 'settings', element: <Settings /> },
			{ path: 'remote', element: <Remote /> },
		],
	},
	{ path: '/login', element: <Login /> },
])

export default function App() {
	return <RouterProvider router={router} />
}


