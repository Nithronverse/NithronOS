import { Link, NavLink, Outlet } from 'react-router-dom'
import { Toasts } from '../ui/toast'
import { Menu } from 'lucide-react'
import { useState } from 'react'

const navItems = [
	{ to: '/', label: 'Dashboard' },
	{ to: '/storage', label: 'Storage' },
	{ to: '/shares', label: 'Shares' },
	{ to: '/apps', label: 'Apps' },
	{ to: '/remote', label: 'Remote' },
	{ to: '/settings', label: 'Settings' },
]

export function Layout() {
	const [open, setOpen] = useState(false)
	return (
		<div className="min-h-screen bg-background text-foreground">
			<header className="sticky top-0 z-40 border-b border-muted/30 bg-background/80 backdrop-blur">
				<div className="mx-auto flex h-14 max-w-6xl items-center justify-between px-4">
					<Link to="/" className="font-semibold">
						NithronOS
					</Link>
					<button className="lg:hidden" onClick={() => setOpen((v) => !v)} aria-label="Menu">
						<Menu className="h-5 w-5" />
					</button>
					<nav className="hidden gap-6 lg:flex">
						{navItems.map((item) => (
							<NavLink
								key={item.to}
								to={item.to}
								className={({ isActive }) =>
									`text-sm ${isActive ? 'text-primary' : 'text-muted-foreground hover:text-foreground'}`
								}
							>
								{item.label}
							</NavLink>
						))}
					</nav>
				</div>
			</header>
			<div className="mx-auto grid max-w-6xl grid-cols-1 gap-6 px-4 py-6 lg:grid-cols-[220px_1fr]">
				<aside className={`space-y-2 ${open ? 'block' : 'hidden'} lg:block`}>
					{navItems.map((item) => (
						<NavLink
							key={item.to}
							to={item.to}
							className={({ isActive }) =>
								`block rounded-md px-3 py-2 text-sm ${
									isActive ? 'bg-card text-primary' : 'text-muted-foreground hover:bg-card hover:text-foreground'
								}`
							}
						>
							{item.label}
						</NavLink>
					))}
				</aside>
				<main>
					<Toasts />
					<Outlet />
				</main>
			</div>
		</div>
	)
}


