import { useEffect, useState } from 'react'
import { api, type RemoteStatus } from '../api/client'

export function Remote() {
	const [status, setStatus] = useState<RemoteStatus | null>(null)
	const [error, setError] = useState<string | null>(null)

	useEffect(() => {
		api.remote.status().then(setStatus).catch((e) => setError(String(e)))
	}, [])

	return (
		<div className="space-y-6">
			<h1 className="text-2xl font-semibold">Remote Access</h1>
			{error && <div className="text-red-400 text-sm">{error}</div>}
			<div className="rounded-lg bg-card p-4">
				<h2 className="mb-2 text-lg font-medium">Current Mode</h2>
				<pre className="text-sm text-muted-foreground">{JSON.stringify(status, null, 2)}</pre>
			</div>
			<div className="rounded-lg bg-card p-4 text-sm text-muted-foreground">
				<p className="mb-2 font-medium text-foreground">Options</p>
				<ul className="list-disc pl-5">
					<li>VPN (WireGuard/Tailscale) — recommended</li>
					<li>Cloudflare Tunnel — no port-forward; requires 2FA</li>
					<li>Direct Port-Forward — enforced 2FA + rate limits</li>
				</ul>
			</div>
		</div>
	)
}


