import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { SettingsUpdates } from '../../pages/SettingsUpdates'
import { toast } from '@/components/ui/toast'
import http from '@/lib/nos-client'

vi.mock('@/components/ui/toast', () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}))

vi.mock('@/lib/nos-client', () => ({
  default: {
    updates: {
      check: vi.fn(),
      apply: vi.fn(),
      rollback: vi.fn(),
    },
    snapshots: {
      recent: vi.fn(),
      prune: vi.fn(),
    },
  },
}))

describe('SettingsUpdates', () => {
  beforeEach(() => { 
    vi.clearAllMocks()
    vi.spyOn(window,'confirm').mockReturnValue(true)
    
    // Default mock setup
    vi.mocked(http.updates.check).mockResolvedValue({ 
      plan: { updates:[{name:'nosd', current:'0.1.0', candidate:'0.2.0'}] }, 
      snapshot_roots: ['/srv'] 
    })
    vi.mocked(http.snapshots.recent).mockResolvedValue([])
  })

  it('renders available updates list', async () => {
    render(<SettingsUpdates />)
    const hdr = await screen.findByText(/Available updates/i)
    expect(hdr).toBeTruthy()
    const row = await screen.findByText(/nosd/i)
    expect(row).toBeTruthy()
  })

  it('disables Apply during request', async () => {
    // Mock the apply to take some time
    let resolveApply: any
    vi.mocked(http.updates.apply).mockImplementation(() => new Promise((resolve) => {
      resolveApply = resolve
    }))
    
    render(<SettingsUpdates />)
    // wait initial load and ensure updates are shown
    await screen.findByText(/Available updates/i)
    await screen.findByText(/nosd/i) // ensure the update is displayed
    
    const btn = screen.getByRole('button', { name: /Apply Updates/i }) as HTMLButtonElement
    expect(btn).toBeTruthy()
    expect(btn.disabled).toBe(false) // button should be enabled initially
    
    fireEvent.click(btn)
    
    // Button should be disabled while applying
    await waitFor(() => {
      const buttons = screen.getAllByRole('button')
      const applyBtn = buttons.find(b => b.textContent?.includes('Apply'))
      expect(applyBtn).toBeTruthy()
      expect((applyBtn as HTMLButtonElement).disabled).toBe(true)
    })
    
    // Resolve the apply promise
    resolveApply({ ok:true, tx_id:'tx-1', snapshots_count:1, updates_count:1 })
    
    // Button should be re-enabled afterwards
    await waitFor(() => {
      const buttons = screen.getAllByRole('button')
      const applyBtn = buttons.find(b => b.textContent?.includes('Apply'))
      expect(applyBtn).toBeTruthy()
      expect((applyBtn as HTMLButtonElement).disabled).toBe(false)
    })
    
    // Verify the toast was called with success message
    await waitFor(() => expect(toast.success).toHaveBeenCalledWith(expect.stringContaining('Updates applied')))
  })

  it('calls rollback API', async () => {
    // Mock to show previous updates
    vi.mocked(http.updates.check).mockResolvedValue({ 
      plan: { updates:[] }, 
      snapshot_roots: ['/srv'] 
    })
    vi.mocked(http.snapshots.recent).mockResolvedValue([
      { tx_id:'tx-1', time: new Date().toISOString(), packages:['nosd'], targets:[], success:true }
    ])
    vi.mocked(http.updates.rollback).mockResolvedValue({ ok:true })
    
    const toastSpy = vi.spyOn(toast, 'success')
    render(<SettingsUpdates />)
    await screen.findByText(/Previous updates/i)
    const rb = await screen.findByRole('button', { name: /Rollback/i })
    fireEvent.click(rb)
    
    await waitFor(() => expect(http.updates.rollback).toHaveBeenCalledWith({ tx_id:'tx-1', confirm:'yes' }))
    await waitFor(() => expect(toastSpy).toHaveBeenCalledWith('Rollback requested'))
  })

  it('prune calls API', async () => {
    vi.mocked(http.snapshots.prune).mockResolvedValue({ ok:true, pruned:{} })
    
    const toastSpy = vi.spyOn(toast, 'success')
    render(<SettingsUpdates />)
    await screen.findByText(/Snapshot retention/i)
    const btn = await screen.findByRole('button', { name: /Prune snapshots now/i })
    fireEvent.click(btn)
    
    await waitFor(() => expect(http.snapshots.prune).toHaveBeenCalledWith({ keep_per_target: 5 }))
    await waitFor(() => expect(toastSpy).toHaveBeenCalledWith('Prune completed'))
  })

  it('shows error on failed apply', async () => {
    vi.mocked(http.updates.apply).mockRejectedValue(new Error('Update failed'))
    const toastErrorSpy = vi.spyOn(toast, 'error')
    
    render(<SettingsUpdates />)
    await screen.findByText(/Available updates/i)
    await screen.findByText(/nosd/i)
    
    const btn = screen.getByRole('button', { name: /Apply Updates/i }) as HTMLButtonElement
    fireEvent.click(btn)
    
    await waitFor(() => expect(toastErrorSpy).toHaveBeenCalledWith('Update failed'))
  })

  it('can toggle snapshot option', async () => {
    render(<SettingsUpdates />)
    await screen.findByText(/Available updates/i)
    
    const checkbox = screen.getByRole('checkbox') as HTMLInputElement
    expect(checkbox.checked).toBe(true)
    
    fireEvent.click(checkbox)
    expect(checkbox.checked).toBe(false)
    
    fireEvent.click(checkbox)
    expect(checkbox.checked).toBe(true)
  })

  it('shows refresh button', async () => {
    render(<SettingsUpdates />)
    const refreshBtn = await screen.findByRole('button', { name: /Refresh/i })
    expect(refreshBtn).toBeTruthy()
    
    // Click refresh and verify APIs are called again
    vi.clearAllMocks()
    fireEvent.click(refreshBtn)
    
    await waitFor(() => expect(http.updates.check).toHaveBeenCalled())
    await waitFor(() => expect(http.snapshots.recent).toHaveBeenCalled())
  })
})