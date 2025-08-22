import { Link, NavLink, Outlet } from 'react-router-dom'
import { Toasts } from '../ui/toast'
import { Bell, Menu, X } from 'lucide-react'
import { useEffect, useState } from 'react'
import api from '@/lib/api'

const navItems = [
	{ to: '/', label: 'Dashboard' },
	{ to: '/storage', label: 'Storage' },
	{ to: '/shares', label: 'Shares' },
	{ to: '/apps', label: 'Apps' },
	{ to: '/remote', label: 'Remote' },
	{ to: '/settings', label: 'Settings' },
	{ to: '/settings/schedules', label: 'Schedules' },
]

export function Layout() {
	const [open, setOpen] = useState(false)
	const [alerts, setAlerts] = useState<any[]>([])
	const [alertsOpen, setAlertsOpen] = useState(false)
	useEffect(() => {
		let stop = false
		async function pull() {
			try {
				const r = await api.health.alerts()
				if (!stop) setAlerts(r.alerts || [])
			} catch {}
			if (!stop) setTimeout(pull, 5000)
		}
		pull()
		return () => { stop = true }
	}, [])
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
					<nav className="hidden items-center gap-6 lg:flex">
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
						<button className="relative" aria-label="Alerts" onClick={() => setAlertsOpen(true)}>
							<Bell className="h-5 w-5 text-muted-foreground" />
							{alerts.length > 0 && <span className="absolute -right-1 -top-1 inline-flex h-4 min-w-[1rem] items-center justify-center rounded-full bg-red-600 px-1 text-[10px] text-white">{alerts.length}</span>}
						</button>
					</nav>
				</div>
			</header>
			<div className="mx-auto grid max-w-6xl grid-cols-1 gap-6 px-4 py-6 lg:grid-cols-[220px_1fr]">
				<aside className={`${open ? 'block' : 'hidden'} space-y-2 lg:block`}>
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
					{alertsOpen && (
						<div className="fixed inset-0 z-50 flex items-start justify-end bg-black/30">
							<div className="mt-14 h-[calc(100%-3.5rem)] w-full max-w-md overflow-auto bg-card p-4 shadow-lg">
								<div className="mb-2 flex items-center justify-between">
									<h2 className="text-lg font-medium">Alerts</h2>
									<button aria-label="Close alerts" onClick={() => setAlertsOpen(false)}><X className="h-4 w-4"/></button>
								</div>
								{alerts.length === 0 ? (
									<div className="text-sm text-muted-foreground">No alerts</div>
								) : (
									<ul className="space-y-2">
										{alerts.map((a:any) => (
											<li key={a.id} className={`rounded border p-2 text-xs ${a.severity==='crit'?'border-red-700 bg-red-950/30':'border-yellow-700 bg-yellow-950/30'}`}>
												<div className="flex items-center justify-between">
													<span className="font-mono">{a.device}</span>
													<span className={`rounded px-1 py-0.5 text-[10px] ${a.severity==='crit'?'bg-red-700 text-white':'bg-yellow-700 text-black'}`}>{a.severity}</span>
												</div>
												<div className="mt-1">{(a.messages||[]).join('; ')}</div>
												<div className="mt-1 text-[10px] text-muted-foreground">{new Date(a.createdAt).toLocaleString()}</div>
											</li>
										))}
									</ul>
								)}
							</div>
						</div>
					)}
					<Outlet />
				</main>
			</div>
		</div>
	)
}


