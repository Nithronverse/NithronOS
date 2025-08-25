import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { SettingsUpdates } from '../../pages/SettingsUpdates'
import * as toast from '@/components/ui/toast'

vi.mock('@/components/ui/toast', () => ({
  pushToast: vi.fn(),
}))

// Deferred gate for the apply call to make the test deterministic
let applyGate: { p: Promise<void>; resolve: () => void } | null = null
function newApplyGate() {
  let resolve!: () => void
  const p = new Promise<void>((r) => (resolve = r))
  applyGate = { p, resolve }
  return applyGate
}
function clearApplyGate() { applyGate = null }

function mockFetchSequence() {
  const original = global.fetch
  const fn = vi.fn().mockImplementation(async (input: RequestInfo, init?: RequestInit) => {
    const url = typeof input === 'string' ? input : input.toString()
    if (url.includes('/api/updates/check')) {
      return new Response(JSON.stringify({ plan: { updates:[{name:'nosd', current:'0.1.0', candidate:'0.2.0'}] }, snapshot_roots: ['/srv'] }), { status:200, headers:{'Content-Type':'application/json'} })
    }
    if (url.includes('/api/snapshots/recent')) {
      return new Response(JSON.stringify([]), { status:200, headers:{'Content-Type':'application/json'} })
    }
    if (url.includes('/api/updates/apply')) {
      // Gate resolution so we can assert the disabled state deterministically
      if (applyGate) {
        await applyGate.p
      } else {
        await new Promise((r) => setTimeout(r, 25))
      }
      return new Response(JSON.stringify({ ok:true, tx_id:'tx-1', snapshots_count:1, updates_count:1 }), { status:200, headers:{'Content-Type':'application/json'} })
    }
    if (url.includes('/api/updates/rollback')) {
      return new Response(JSON.stringify({ ok:true }), { status:200, headers:{'Content-Type':'application/json'} })
    }
    if (url.includes('/api/snapshots/prune')) {
      return new Response(JSON.stringify({ ok:true, pruned:{} }), { status:200, headers:{'Content-Type':'application/json'} })
    }
    return original(input, init)
  })
  // @ts-ignore
  global.fetch = fn
  return () => { global.fetch = original }
}

describe('SettingsUpdates', () => {
  let restore: (()=>void) | undefined
  beforeEach(() => { restore?.(); restore = mockFetchSequence(); vi.spyOn(window,'confirm').mockReturnValue(true) })

  it('renders available updates list', async () => {
    render(<SettingsUpdates />)
    const hdr = await screen.findByText(/Available updates/i)
    expect(hdr).toBeTruthy()
    const row = await screen.findByText(/nosd/i)
    expect(row).toBeTruthy()
  })

  it('disables Apply during request', async () => {
    render(<SettingsUpdates />)
    // wait initial load and ensure updates are shown
    await screen.findByText(/Available updates/i)
    await screen.findByText(/nosd/i) // ensure the update is displayed
    
    const btn = screen.getByRole('button', { name: /Apply Updates/i }) as HTMLButtonElement
    expect(btn).toBeTruthy()
    expect(btn.disabled).toBe(false) // button should be enabled initially
    
    const gate = newApplyGate()
    
    // Clear any previous calls from initial load
    if ((global.fetch as any).mockClear) {
      (global.fetch as any).mockClear()
    }
    
    fireEvent.click(btn)
    
    // goes into applying state (allow async state update); re-query by role to handle text change
    await waitFor(() => {
      const buttons = screen.getAllByRole('button')
      const applyBtn = buttons.find(b => b.textContent?.includes('Apply'))
      expect(applyBtn).toBeTruthy()
      expect((applyBtn as HTMLButtonElement).disabled).toBe(true)
    }, { timeout: 2000 })
    
    // ensure apply API was called (search mock.calls for the apply URL)
    await waitFor(() => {
      const fetchMock = global.fetch as any
      const calls = fetchMock.mock?.calls || []
      const hasApplyCall = calls.some((args: any[]) => {
        const url = args[0]
        return typeof url === 'string' && url.includes('/api/updates/apply')
      })
      expect(hasApplyCall).toBe(true)
    }, { timeout: 3000 })
    
    // release the gate so apply resolves
    gate.resolve()
    
    // and button should be re-enabled afterwards (re-query)
    await waitFor(() => {
      const buttons = screen.getAllByRole('button')
      const applyBtn = buttons.find(b => b.textContent?.includes('Apply'))
      expect(applyBtn).toBeTruthy()
      expect((applyBtn as HTMLButtonElement).disabled).toBe(false)
    }, { timeout: 2000 })
    
    clearApplyGate()
  })

  it('calls rollback API', async () => {
    // Modify recent to include one tx
    const original = global.fetch
    // @ts-ignore
    global.fetch = vi.fn(async (input: RequestInfo, init?: RequestInit) => {
      const url = typeof input === 'string' ? input : input.toString()
      if (url.includes('/api/updates/check')) {
        return new Response(JSON.stringify({ plan: { updates:[] }, snapshot_roots: ['/srv'] }), { status:200, headers:{'Content-Type':'application/json'} })
      }
      if (url.includes('/api/snapshots/recent')) {
        return new Response(JSON.stringify([{ tx_id:'tx-1', time: new Date().toISOString(), packages:['nosd'], targets:[], success:true }]), { status:200, headers:{'Content-Type':'application/json'} })
      }
      if (url.includes('/api/updates/rollback')) {
        return new Response(JSON.stringify({ ok:true }), { status:200, headers:{'Content-Type':'application/json'} })
      }
      return new Response(JSON.stringify({}), { status:200, headers:{'Content-Type':'application/json'} })
    })
    const toastSpy = vi.spyOn(toast, 'pushToast')
    render(<SettingsUpdates />)
    await screen.findByText(/Previous updates/i)
    const rb = await screen.findByRole('button', { name: /Rollback/i })
    fireEvent.click(rb)
    await waitFor(() => expect((global.fetch as any)).toHaveBeenCalled())
    await waitFor(()=> {
      expect(toastSpy).toHaveBeenCalled()
    })
    global.fetch = original
  })
})


