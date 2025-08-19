import { createBrowserRouter, RouterProvider } from 'react-router-dom'
import { Layout } from './components/layouts/Layout'
import { Dashboard } from './pages/Dashboard'
import { Storage } from './pages/Storage'
import { Shares } from './pages/Shares'
import { Apps } from './pages/Apps'
import { Settings } from './pages/Settings'
import { Remote } from './pages/Remote'

const router = createBrowserRouter([
	{
		path: '/',
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
])

export default function App() {
	return <RouterProvider router={router} />
}


