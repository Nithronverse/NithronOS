import { useState } from 'react'
import { apiPost } from '../api/http'
import { useNavigate } from 'react-router-dom'
import { pushToast } from '@/components/ui/toast'
import BrandHeader from '@/components/BrandHeader'

export function Login() {
	const [username, setUsername] = useState('')
	const [password, setPassword] = useState('')
	const [remember, setRemember] = useState(false)
	const [needCode, setNeedCode] = useState(false)
	const [code, setCode] = useState('')
	const [error, setError] = useState<string | null>(null)
	const nav = useNavigate()

	async function onSubmit(e: React.FormEvent) {
		e.preventDefault()
		setError(null)
		try {
			const body: any = { username, password, rememberMe: remember }
			if (needCode && code.trim()) body.code = code.trim().replace(/\s+/g,'')
			await apiPost<{ ok: boolean }>('/api/auth/login', body)
			nav('/')
		} catch (e: any) {
			const msg = String(e?.message || e)
			// Throttling (rate limit)
			if (msg.includes('429') || /try again later/i.test(msg)) {
				pushToast('Too many attempts. Please try again later.', 'error')
				setError('Too many attempts. Please try again later.')
				return
			}
			// If code required, reveal TOTP/recovery input
			if (/code required/i.test(msg)) {
				setNeedCode(true)
				pushToast('Enter your 6-digit TOTP or a recovery code.', 'error')
				setError('Two-factor code required')
				return
			}
			// Generic unauthorized / lockout
			if (msg.includes('401')) {
				pushToast('Invalid credentials or account temporarily locked.', 'error')
				setError('Invalid credentials or account temporarily locked.')
				return
			}
			setError(msg)
			pushToast(msg, 'error')
		}
	}

	return (
		<div className="min-h-screen w-full flex items-center justify-center">
			<div className="relative w-full max-w-sm p-6 pt-20">
				<BrandHeader />
				<h1 className="mb-4 text-center text-2xl font-semibold">Sign in</h1>
				{error && <div className="mb-3 text-sm text-red-400">{error}</div>}
				<form onSubmit={onSubmit} className="space-y-3">
					<input className="w-full rounded bg-card p-2" placeholder="Username" value={username} onChange={(e) => setUsername(e.target.value)} />
					<input className="w-full rounded bg-card p-2" placeholder="Password" type="password" value={password} onChange={(e) => setPassword(e.target.value)} />
					<label className="flex items-center gap-2 text-sm">
						<input type="checkbox" checked={remember} onChange={(e)=>setRemember(e.target.checked)} /> Remember me
					</label>
					{needCode && (
						<input className="w-full rounded bg-card p-2 tracking-widest" placeholder="TOTP or recovery code" value={code} onChange={(e) => setCode(e.target.value)} />
					)}
					<button className="btn bg-primary text-primary-foreground block w-1/2 mx-auto py-3" type="submit">
						{needCode ? 'Verify and Sign in' : 'Sign in'}
					</button>
				</form>
			</div>
		</div>
	)
}


