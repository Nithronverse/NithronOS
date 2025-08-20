import { useEffect, useState } from 'react'
import { api } from '../api/client'
import QRCode from 'qrcode'

export function Settings() {
	const [me, setMe] = useState<any | null>(null)
	const [loading, setLoading] = useState(true)
	const [setup, setSetup] = useState<{ uri: string } | null>(null)
	const [qr, setQr] = useState<string>('')
	const [code, setCode] = useState('')
	const [error, setError] = useState<string | null>(null)
	const [downloading, setDownloading] = useState(false)

	useEffect(() => {
		api.auth.me().then(setMe).finally(() => setLoading(false))
	}, [])

	async function startTotp() {
		setError(null)
		try {
			const res = await fetch('/api/auth/totp/setup', { method: 'POST', headers: { 'X-CSRF-Token': getCSRF() } })
			if (!res.ok) throw new Error(`HTTP ${res.status}`)
			const j = await res.json()
			setSetup({ uri: j?.otpauth_uri || '' })
			try {
				setQr(await QRCode.toDataURL(j?.otpauth_uri || '', { margin: 1, width: 192 }))
			} catch {}
		} catch (e: any) {
			setError(e?.message || String(e))
		}
	}

	async function confirmTotp() {
		setError(null)
		try {
			const res = await fetch('/api/auth/totp/confirm', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': getCSRF() },
				body: JSON.stringify({ code }),
			})
			if (!res.ok) throw new Error(`HTTP ${res.status}`)
			setSetup(null)
			setQr('')
			setMe({ ...(me || {}), totp_enabled: true })
			setCode('')
		} catch (e: any) {
			setError(e?.message || String(e))
		}
	}

	function getCSRF(): string {
		const m = document.cookie.match(/(?:^|; )nos_csrf=([^;]*)/)
		return m ? decodeURIComponent(m[1]) : ''
	}

	async function downloadBundle() {
		setDownloading(true)
		try {
			const blob = await api.support.bundle()
			const url = URL.createObjectURL(blob)
			const a = document.createElement('a')
			a.href = url
			a.download = 'nos-support-bundle.tar.gz'
			document.body.appendChild(a)
			a.click()
			a.remove()
			URL.revokeObjectURL(url)
		} catch (e: any) {
			try { const { pushToast } = await import('../components/ui/toast'); pushToast(`Download failed: ${e?.message || e}`, 'error') } catch {}
		} finally {
			setDownloading(false)
		}
	}

	return (
		<div className="space-y-6">
			<h1 className="text-2xl font-semibold">Settings</h1>
			<div className="rounded-lg bg-card p-4 space-y-3">
				<h2 className="text-lg font-medium">Profile</h2>
				{loading ? (
					<div className="text-sm text-muted-foreground">Loading…</div>
				) : (
					<div className="text-sm">
						<div>
							Email: <span className="text-muted-foreground">{me?.email}</span>
						</div>
						<div>
							Roles: <span className="text-muted-foreground">{Array.isArray(me?.roles) ? me.roles.join(', ') : ''}</span>
						</div>
						<div className="mt-2">
							2FA (TOTP): {me?.totp_enabled ? (
								<span className="text-green-400">Enabled</span>
							) : (
								<span className="text-yellow-300">Disabled</span>
							)}
						</div>
						{!me?.totp_enabled && !setup && (
							<button className="mt-2 rounded bg-primary px-3 py-1 text-sm text-background" onClick={startTotp}>
								Start TOTP setup
							</button>
						)}
						{setup && (
							<div className="mt-3 space-y-2">
								<div className="text-muted-foreground">Scan in your authenticator app:</div>
								{qr ? (
									<img src={qr} alt="TOTP QR" className="h-44 w-44 rounded bg-white p-2" />
								) : (
									<div className="rounded border border-muted/30 bg-background p-2 text-xs break-all">{setup.uri}</div>
								)}
								<div>
									<label className="block text-sm">Enter code</label>
									<input
										className="mt-1 w-full rounded border border-muted/30 bg-background px-2 py-1"
										value={code}
										onChange={(e) => setCode(e.target.value.replace(/\D/g, ''))}
										maxLength={6}
										inputMode="numeric"
									/>
								</div>
								<div className="flex gap-2">
									<button className="rounded border border-muted/30 px-3 py-1 text-sm" onClick={() => { setSetup(null); setQr('') }}>
										Cancel
									</button>
									<button className="rounded bg-primary px-3 py-1 text-sm text-background" onClick={confirmTotp} disabled={code.length !== 6}>
										Confirm
									</button>
								</div>
							</div>
						)}
						{error && <div className="rounded border border-red-500/30 bg-red-500/10 p-2 text-red-400 text-xs">{error}</div>}
					</div>
				)}
			</div>

			<div className="rounded-lg bg-card p-4 space-y-3">
				<h2 className="text-lg font-medium">System</h2>
				<p className="text-sm text-muted-foreground">Download a support bundle with logs and diagnostics (redacted).</p>
				<button
					className="inline-block rounded bg-primary px-3 py-1 text-sm text-background disabled:opacity-50"
					onClick={downloadBundle}
					disabled={downloading}
				>
					{downloading ? 'Preparing…' : 'Download Support Bundle'}
				</button>
			</div>
		</div>
	)
}


