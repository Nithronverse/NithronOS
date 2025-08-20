import { useEffect, useState } from 'react'
import { api, type RemoteStatus } from '../api/client'
import { Modal } from '../components/ui/modal'

export function Remote() {
	const [status, setStatus] = useState<RemoteStatus | null>(null)
	const [fw, setFw] = useState<any | null>(null)
	const [mode, setMode] = useState<string>('lan-only')
	const [planning, setPlanning] = useState(false)
	const [plan, setPlan] = useState<string>('')
	const [applying, setApplying] = useState(false)
	const [rollingBack, setRollingBack] = useState(false)
	const [showConfirm, setShowConfirm] = useState(false)
	const [showPlan, setShowPlan] = useState(false)
	const [totp, setTotp] = useState('')
	const [has2FA, setHas2FA] = useState<boolean | null>(null)
	const [error, setError] = useState<string | null>(null)

	useEffect(() => {
		api.remote.status()
			.then((s) => {
				setStatus(s)
				setMode(s.mode)
			})
			.catch((e) => setError(String(e)))
		api.firewall.status().then(setFw).catch(() => {})
		api.auth.me().then((u) => setHas2FA(!!(u as any)?.totp_enabled)).catch(() => {})
	}, [])

	async function doPlan() {
		setPlanning(true)
		try {
			const res: any = await api.firewall.plan(mode)
			setPlan(res?.rules || '')
			setShowPlan(true)
		} catch (e: any) {
			setError(e?.message ? String(e.message) : String(e))
		} finally {
			setPlanning(false)
		}
	}

	async function doApply() {
		setApplying(true)
		try {
			await api.firewall.apply(mode, mode !== 'lan-only' ? totp : undefined)
			setPlan('')
			// success toast
			try { const { pushToast } = await import('../components/ui/toast'); pushToast('Firewall applied'); } catch {}
			const s = await api.remote.status()
			setStatus(s)
			const f = await api.firewall.status()
			setFw(f)
		} catch (e: any) {
			setError(e?.message ? String(e.message) : String(e))
			try { const { pushToast } = await import('../components/ui/toast'); pushToast(`Failed: ${e?.message || e}`, 'error'); } catch {}
		} finally {
			setApplying(false)
			setShowConfirm(false)
			setTotp('')
		}
	}

	async function doRollback() {
		setRollingBack(true)
		try {
			await api.firewall.rollback()
			try { const { pushToast } = await import('../components/ui/toast'); pushToast('Rollback requested') } catch {}
			const s = await api.remote.status()
			setStatus(s)
			const f = await api.firewall.status()
			setFw(f)
		} catch (e: any) {
			try { const { pushToast } = await import('../components/ui/toast'); pushToast(`Rollback failed: ${e?.message || e}`, 'error') } catch {}
		} finally {
			setRollingBack(false)
		}
	}

	return (
		<>
			<div className="space-y-6">
				<h1 className="text-2xl font-semibold">Remote Access</h1>
				<div className="rounded border border-muted/30 bg-card/60 p-2 text-xs flex flex-wrap gap-3">
					<span>
						<span
							className={`mr-1 inline-block h-2 w-2 rounded-full ${status?.mode === 'lan-only' ? 'bg-green-500' : 'bg-yellow-500'}`}
						/>
						Mode: {status?.mode ?? '…'}
					</span>
					<span>nft: {fw ? (fw.nft_present ? 'yes' : 'no') : '…'}</span>
					<span>ufw: {fw ? (fw.ufw_present ? 'yes' : 'no') : '…'}</span>
					<span>firewalld: {fw ? (fw.firewalld_present ? 'yes' : 'no') : '…'}</span>
				</div>
				{error && <div className="rounded border border-red-500/30 bg-red-500/10 p-2 text-red-400 text-sm">{error}</div>}
				<div className="rounded-lg bg-card p-4">
					<h2 className="mb-2 text-lg font-medium">Current Mode</h2>
					<pre className="text-sm text-muted-foreground">{JSON.stringify(status, null, 2)}</pre>
				</div>
				<div className="rounded-lg bg-card p-4 space-y-3">
					<div>
						<label className="block text-sm mb-1">Mode</label>
						<select
							className="bg-background border border-muted/30 rounded px-2 py-1 text-sm"
							value={mode}
							onChange={(e) => setMode(e.target.value)}
						>
							<option value="lan-only">LAN-only (default)</option>
							<option value="vpn-only">VPN-only</option>
							<option value="tunnel">Tunnel (Cloudflare)</option>
							<option value="direct">Direct (Public)</option>
						</select>
					</div>
					<div className="flex flex-wrap gap-2">
						<button
							className="rounded bg-primary px-3 py-1 text-sm text-background disabled:opacity-50"
							onClick={doPlan}
							disabled={planning}
						>
							Plan
						</button>
						<button
							className="rounded bg-primary/80 px-3 py-1 text-sm text-background disabled:opacity-50"
							onClick={() => {
								if (mode !== 'lan-only') setShowConfirm(true)
								else doApply()
							}}
							disabled={applying || !plan}
						>
							Apply
						</button>
						<button
							className="rounded border border-muted/30 px-3 py-1 text-sm disabled:opacity-50"
							onClick={async () => { await doRollback() }}
							disabled={rollingBack}
						>
							Rollback
						</button>
					</div>
					{has2FA === false && (
						<div className="text-xs text-yellow-300">
							2FA is not enabled on your account. Enable it in{' '}
							<a href="/settings" className="underline">Settings → Profile</a>{' '}before enabling remote modes.
						</div>
					)}
					{/* plan is now shown in a modal */}
					{fw && (
						<div className="text-xs text-muted-foreground">
							nft: {String(fw.nft_present)} | ufw: {String(fw.ufw_present)} | firewalld: {String(fw.firewalld_present)}
						</div>
					)}
				</div>
			</div>
			<Modal
				open={showPlan}
				title="Planned rules"
				onClose={() => setShowPlan(false)}
				footer={
					<>
						<button className="rounded border border-muted/30 px-3 py-1 text-sm" onClick={() => setShowPlan(false)}>Close</button>
						<button
							className="rounded bg-primary px-3 py-1 text-sm text-background"
							onClick={() => { setShowPlan(false); if (mode !== 'lan-only') setShowConfirm(true); else doApply() }}
						>
							Proceed to Apply
						</button>
					</>
				}
			>
				<pre className="text-xs text-muted-foreground overflow-auto max-h-80 whitespace-pre-wrap">{plan}</pre>
			</Modal>
			<Modal
				open={showConfirm}
				title="Confirm change"
				onClose={() => setShowConfirm(false)}
				footer={
					<>
						<button className="rounded border border-muted/30 px-3 py-1 text-sm" onClick={() => setShowConfirm(false)}>
							Cancel
						</button>
						<button
							className="rounded bg-primary px-3 py-1 text-sm text-background disabled:opacity-50"
							onClick={doApply}
							disabled={applying || (mode !== 'lan-only' && totp.length !== 6)}
						>
							Confirm & Apply
						</button>
					</>
				}
			>
				<div className="space-y-3">
					<p>
						You are switching to <span className="font-medium">{mode}</span>. Non–lan-only modes require 2FA.
					</p>
					{mode === 'direct' && (
						<div className="rounded border border-yellow-500/30 bg-yellow-500/10 p-2 text-yellow-300 text-xs">
							<strong>Warning:</strong> Direct mode exposes the web UI to the Internet. Ensure strong passwords, 2FA is enabled, and rate limiting is configured.
						</div>
					)}
					<label className="block text-sm">
						TOTP Code
						<input
							type="text"
							inputMode="numeric"
							maxLength={6}
							className="mt-1 w-full rounded border border-muted/30 bg-background px-2 py-1"
							value={totp}
							onChange={(e) => setTotp(e.target.value.replace(/\D/g, ''))}
						/>
					</label>
				</div>
			</Modal>
		</>
	)
}


