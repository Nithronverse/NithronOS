import { useState } from 'react'
import { apiPost } from '../api/http'
import { useNavigate } from 'react-router-dom'

export function Login() {
	const [email, setEmail] = useState('admin@example.com')
	const [password, setPassword] = useState('admin123')
	const [needTotp, setNeedTotp] = useState(false)
	const [totp, setTotp] = useState('')
	const [error, setError] = useState<string | null>(null)
	const nav = useNavigate()

	async function onSubmit(e: React.FormEvent) {
		e.preventDefault()
		setError(null)
		try {
			const body: any = { email, password }
			if (needTotp) body.totp = totp
			const res = await apiPost<{ ok?: boolean; need_totp?: boolean }>('/api/auth/login', body)
			if (res.need_totp) {
				setNeedTotp(true)
				return
			}
			nav('/')
		} catch (e: any) {
			setError(String(e?.message || e))
		}
	}

	return (
		<div className="mx-auto max-w-sm p-6">
			<h1 className="mb-4 text-2xl font-semibold">Sign in</h1>
			{error && <div className="mb-3 text-sm text-red-400">{error}</div>}
			<form onSubmit={onSubmit} className="space-y-3">
				<input className="w-full rounded bg-card p-2" placeholder="Email" value={email} onChange={(e) => setEmail(e.target.value)} />
				<input className="w-full rounded bg-card p-2" placeholder="Password" type="password" value={password} onChange={(e) => setPassword(e.target.value)} />
				{needTotp && (
					<input className="w-full rounded bg-card p-2" placeholder="TOTP Code" value={totp} onChange={(e) => setTotp(e.target.value)} />
				)}
				<button className="btn bg-primary text-primary-foreground w-full" type="submit">
					{needTotp ? 'Verify' : 'Sign in'}
				</button>
			</form>
		</div>
	)
}


