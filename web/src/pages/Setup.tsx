import { useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useApi } from '@/lib/useApi'
import { pushToast } from '@/components/ui/toast'
import { useForm } from 'react-hook-form'
import { z } from 'zod'
import { zodResolver } from '@hookform/resolvers/zod'
import QRCode from 'qrcode'
import BrandHeader from '@/components/BrandHeader'
import { useGlobalNotice } from '@/lib/globalNotice'

type SetupState = { firstBoot: boolean; otpRequired: boolean }

type Creds = { username: string; password: string }

export default function Setup() {
  const nav = useNavigate()
  const { get, post, postAuth } = useApi()
  const [step, setStep] = useState<1|2|3|4>(1)
  const [error, setLocalError] = useState<string|undefined>()
  const [token, setToken] = useState<string|undefined>() // setup token (memory only)
  const [creds, setCreds] = useState<Creds | undefined>() // memory-only credentials for TOTP login
  const { notice } = useGlobalNotice()

  useEffect(() => {
    (async () => {
      // Clear any stale cookies server-side; ignore failures
      try { await fetch('/api/auth/logout', { method: 'POST', credentials: 'include' }) } catch {}
      try {
        const st = await get<SetupState>('/api/setup/state')
        if (!st.firstBoot) {
          setLocalError('Setup already completed. Sign in.')
        }
      } catch (e: any) {
        if (e && e.status === 410 && e.code === 'setup.complete') {
          setLocalError('Setup already completed. Sign in.')
        } else if (e && e.status === 404) {
          setLocalError('Setup endpoint not available.')
        } else {
          setLocalError(String(e?.message || e))
        }
      }
    })()
  }, [])

  return (
    <div className="min-h-screen w-full flex items-center justify-center">
      <div className="relative w-full max-w-md p-6 pt-20">
        <BrandHeader />
        <h1 className="mb-4 text-center text-2xl font-semibold">First-time Setup</h1>
        {(notice && notice.title.includes('Backend unreachable')) && (
          <div className="mb-4 text-center text-red-400 text-sm">
            The server didn’t return JSON for /api/*. Check reverse proxy config.
          </div>
        )}
        {error && (
          <div className="mb-4 text-center text-yellow-400">
            {error}
            {error.includes('Setup already completed') && (
              <div className="mt-2">
                <button className="btn bg-primary text-primary-foreground" onClick={() => nav('/login')}>Sign In</button>
              </div>
            )}
          </div>
        )}
        <Steps step={step} />
        {step === 1 && <StepOTP onVerified={(t) => { setToken(t); setStep(2) }} setError={setLocalError} setLoading={() => {}} post={post} disabled={!!notice} />}
        {step === 2 && <StepCreateAdmin token={token!} onDone={(needsTotp, c) => {
          if (needsTotp && c) { setCreds(c); setStep(3) } else { setStep(4) }
        }} setError={setLocalError} setLoading={() => {}} postAuth={postAuth} disabled={!!notice} />}
        {step === 3 && <StepTOTPEnroll creds={creds!} onDone={() => setStep(4)} disabled={!!notice} />}
        {step === 4 && <StepDone onGoLogin={() => nav('/login')} disabled={!!notice} />}
      </div>
    </div>
  )
}

function Steps({ step }: { step: number }) {
  const items = useMemo(() => [
    { n: 1, t: 'Verify OTP' },
    { n: 2, t: 'Create Admin' },
    { n: 3, t: 'Enable 2FA (optional)' },
    { n: 4, t: 'Finish' },
  ], [])
  return (
    <div className="flex items-center justify-between mb-6">
      {items.map(it => (
        <div key={it.n} className={`text-sm ${step >= it.n ? 'text-primary' : 'text-muted-foreground'}`}>{it.n}. {it.t}</div>
      ))}
    </div>
  )
}

function StepOTP({ onVerified, setError, setLoading, post, disabled }: { onVerified: (token: string) => void; setError: (s?: string)=>void; setLoading: (b: boolean)=>void; post: <T>(path: string, body?: any) => Promise<T>; disabled?: boolean }) {
  const [otp, setOtp] = useState('')
  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setError(undefined)
    if (otp.replace(/\s+/g, '').length !== 6) { const m='Enter the 6-digit code'; setError(m); pushToast(m,'error'); return }
    setLoading(true)
    try {
      const res = await post<{ ok: boolean; token: string }>("/api/setup/verify-otp", { otp: otp.replace(/\s+/g, '') })
      onVerified(res.token)
    } catch (err: any) {
      let m = String(err?.message || err)
      if (err && (err.status === 401 || err.status === 403)) {
        m = 'OTP expired/invalid. Regenerate and try again.'
      }
      if (err && err.status === 429) {
        const sec = typeof err.retryAfterSec === 'number' ? err.retryAfterSec : undefined
        m = `Too many attempts. Try again${sec ? ` in ${sec}s` : ''}.`
      }
      setError(m)
      pushToast(m, 'error')
    } finally {
      setLoading(false)
    }
  }
  return (
    <form onSubmit={submit} className="space-y-3">
      <label htmlFor="otp" className="block text-sm">One-time OTP (6 digits)</label>
      <input id="otp" className="w-full rounded bg-card p-2 tracking-widest" placeholder="123 456" value={otp} onChange={(e) => setOtp(e.target.value)} aria-label="One-time OTP (6 digits)" disabled={disabled} title={disabled? 'Waiting for backend': undefined} />
      <button className="btn bg-primary text-primary-foreground block w-1/2 mx-auto py-3" type="submit" disabled={disabled} title={disabled? 'Waiting for backend': undefined}>Verify</button>
    </form>
  )
}

const passwordStrength = (p: string) => {
  let score = 0
  if (p.length >= 12) score++
  if (/[a-z]/.test(p)) score++
  if (/[A-Z]/.test(p)) score++
  if (/[0-9]/.test(p)) score++
  if (/[^A-Za-z0-9]/.test(p)) score++
  if (score > 4) score = 4
  return score
}

const CreateAdminSchema = z.object({
  username: z.string().regex(/^[a-z0-9_-]{3,32}$/,{ message:'3–32 chars: lowercase letters, digits, - or _' }),
  password: z.string().min(12, { message:'Min 12 characters' }).refine((v)=>passwordStrength(v) >= 3, { message:'Use a mix of cases, digits, symbols' }),
  confirm: z.string(),
  enableTotp: z.boolean().optional(),
}).refine((data) => data.password === data.confirm, { message:'Passwords do not match', path:['confirm'] })

type CreateAdminInput = z.infer<typeof CreateAdminSchema>

function StepCreateAdmin({ token, onDone, setError, setLoading, postAuth, disabled }: { token: string; onDone: (needsTotp: boolean, creds?: Creds) => void; setError: (s?: string)=>void; setLoading: (b: boolean)=>void; postAuth: <T>(path: string, token: string, body?: any) => Promise<T>; disabled?: boolean }) {
  const { register, handleSubmit, formState: { errors, isSubmitting }, watch } = useForm<CreateAdminInput>({
    resolver: zodResolver(CreateAdminSchema),
    defaultValues: { username:'', password:'', confirm:'', enableTotp:false },
    mode: 'onChange',
  })
  const pwd = watch('password') || ''
  const strength = passwordStrength(pwd)

  const onSubmit = async (data: CreateAdminInput) => {
    setError(undefined)
    setLoading(true)
    try {
      await postAuth<{ ok: boolean }>("/api/setup/create-admin", token, { username: data.username, password: data.password, enable_totp: !!data.enableTotp })
      onDone(!!data.enableTotp, data.enableTotp ? { username: data.username, password: data.password } : undefined)
    } catch (err: any) {
      let m = String(err?.message || err)
      if (err && (err.status === 401 || err.status === 403)) {
        m = 'OTP expired/invalid. Regenerate and try again.'
      }
      if (err && err.status === 429) {
        const sec = typeof err.retryAfterSec === 'number' ? err.retryAfterSec : undefined
        m = `Too many attempts. Try again${sec ? ` in ${sec}s` : ''}.`
      }
      setError(m)
      pushToast(m, 'error')
    } finally {
      setLoading(false)
    }
  }

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-3 mt-6">
      <label className="block text-sm">Username</label>
      <input className="w-full rounded bg-card p-2" placeholder="username" {...register('username')} disabled={disabled} title={disabled? 'Waiting for backend': undefined} />
      {errors.username && <div className="text-xs text-red-400">{errors.username.message as string}</div>}

      <label className="block text-sm">Password</label>
      <input className="w-full rounded bg-card p-2" type="password" {...register('password')} disabled={disabled} title={disabled? 'Waiting for backend': undefined} />
      <PasswordMeter strength={strength} />
      {errors.password && <div className="text-xs text-red-400">{errors.password.message as string}</div>}

      <label className="block text-sm">Confirm password</label>
      <input className="w-full rounded bg-card p-2" type="password" {...register('confirm')} disabled={disabled} title={disabled? 'Waiting for backend': undefined} />
      {errors.confirm && <div className="text-xs text-red-400">{errors.confirm.message as string}</div>}

      <label className="flex items-center gap-2 text-sm"><input type="checkbox" {...register('enableTotp')} disabled={disabled} /> Enable 2FA now</label>
      <button className="btn bg-primary text-primary-foreground block w-1/2 mx-auto py-3" type="submit" disabled={isSubmitting || disabled} title={disabled? 'Waiting for backend': undefined}>Create Admin</button>
    </form>
  )
}

function PasswordMeter({ strength }: { strength: number }) {
  const colors = ['bg-red-500','bg-orange-500','bg-yellow-500','bg-green-500']
  const labels = ['Very weak','Weak','Okay','Strong']
  const idx = Math.max(0, Math.min(3, strength-1))
  return (
    <div className="mt-1">
      <div className="h-1 w-full bg-muted rounded">
        <div className={`h-1 rounded ${strength>0?colors[idx]:'bg-red-500'}`} style={{ width: `${Math.max(10, strength*25)}%` }}></div>
      </div>
      <div className="text-[11px] text-muted-foreground mt-1">{strength>0?labels[idx]:'Very weak'}</div>
    </div>
  )
}

function StepTOTPEnroll({ creds, onDone, disabled }: { creds: Creds; onDone: ()=>void; disabled?: boolean }) {
  const { post, get } = useApi()
  const [otpauth, setOtpauth] = useState<string>('')
  const [qr, setQr] = useState<string>('')
  const [code, setCode] = useState('')
  const [recovery, setRecovery] = useState<string[] | null>(null)

  useEffect(() => {
    (async () => {
      try {
        await post<any>('/api/auth/login', { username: creds.username, password: creds.password })
      } catch {}
      try {
        const resp = await get<{ otpauth_url: string; qr_png_base64?: string }>("/api/auth/totp/enroll")
        setOtpauth(resp.otpauth_url)
        if (resp.qr_png_base64 && resp.qr_png_base64.length > 0) {
          setQr(`data:image/png;base64,${resp.qr_png_base64}`)
        } else if (resp.otpauth_url) {
          const dataUrl = await QRCode.toDataURL(resp.otpauth_url)
          setQr(dataUrl)
        }
      } catch (e:any) {
        pushToast(String(e?.message||e), 'error')
      }
    })()
  }, [])

  const manualSecret = (() => {
    try {
      const u = new URL(otpauth)
      return u.searchParams.get('secret') || ''
    } catch { return '' }
  })()

  async function verify(e: React.FormEvent) {
    e.preventDefault()
    if (code.replace(/\s+/g,'').length !== 6) { pushToast('Enter the 6-digit code','error'); return }
    try {
      const resp = await post<{ ok: boolean; recovery_codes: string[] }>("/api/auth/totp/verify", { code: code.replace(/\s+/g,'') })
      setRecovery(resp.recovery_codes || [])
    } catch (e:any) {
      pushToast(String(e?.message||e), 'error')
    }
  }

  function copyCodes() {
    if (!recovery) return
    navigator.clipboard.writeText(recovery.join('\n')).then(()=>pushToast('Copied to clipboard','success'))
  }
  function downloadCodes() {
    if (!recovery) return
    const blob = new Blob([recovery.join('\n')+'\n'], { type:'text/plain' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = 'nithronos-recovery-codes.txt'
    document.body.appendChild(a)
    a.click()
    a.remove()
    URL.revokeObjectURL(url)
  }

  return (
    <div className="mt-6">
      {!recovery ? (
        <>
          <h2 className="text-lg font-semibold mb-3">Enable 2FA (TOTP)</h2>
          {qr && <img src={qr} alt="QR" className="mx-auto mb-3 h-40" />}
          {manualSecret && (
            <div className="text-sm text-center mb-3">
              Manual code: <span className="font-mono">{manualSecret}</span>
            </div>
          )}
          <form onSubmit={verify} className="space-y-3">
            <label htmlFor="totp" className="block text-sm">TOTP code</label>
            <input id="totp" className="w-full rounded bg-card p-2 tracking-widest" placeholder="TOTP code" value={code} onChange={(e)=>setCode(e.target.value)} aria-label="TOTP code" disabled={disabled} title={disabled? 'Waiting for backend': undefined} />
            <button className="btn bg-primary text-primary-foreground w-full" type="submit" disabled={disabled} title={disabled? 'Waiting for backend': undefined}>Verify</button>
          </form>
        </>
      ) : (
        <div className="text-center">
          <h2 className="text-lg font-semibold mb-3">Recovery Codes</h2>
          <p className="text-sm mb-3">Store these codes safely. Each can be used once.</p>
          <div className="bg-card rounded p-3 text-left font-mono text-sm max-h-48 overflow-auto">
            {recovery.map((c, i)=> <div key={i}>{c}</div>)}
          </div>
          <div className="flex gap-2 justify-center mt-3">
            <button className="btn bg-secondary" onClick={copyCodes}>Copy</button>
            <button className="btn bg-secondary" onClick={downloadCodes}>Download</button>
            <button className="btn bg-primary text-primary-foreground" onClick={onDone}>Continue</button>
          </div>
        </div>
      )}
    </div>
  )
}

function StepDone({ onGoLogin, disabled }: { onGoLogin: ()=>void; disabled?: boolean }) {
  return (
    <div className="text-center mt-6">
      <h2 className="text-lg font-semibold mb-2">Setup Complete</h2>
      <p className="text-sm mb-4">Your admin account is ready. You can now sign in.</p>
      <button className="btn bg-primary text-primary-foreground block w-1/2 mx-auto py-3" onClick={onGoLogin} disabled={disabled} title={disabled? 'Waiting for backend': undefined}>Go to Sign in</button>
    </div>
  )
}


