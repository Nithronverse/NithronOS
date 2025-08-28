import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { toast } from '@/components/ui/toast'
import { api } from '@/lib/api-client'
import { PoolDetails } from '../PoolDetails'

vi.mock('@/lib/api-client', () => ({
  api: {
    pools: {
      getMountOptions: vi.fn(),
      updateMountOptions: vi.fn(),
      list: vi.fn(),
      scrub: vi.fn(),
      balance: vi.fn(),
      trim: vi.fn(),
    },
  },
}))

vi.mock('@/components/ui/toast', () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}))

describe('PoolMountOptions', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.spyOn(window, 'fetch').mockResolvedValue(new Response(JSON.stringify([{ id:'p1', mount:'/mnt/p1', uuid:'U', raid:'raid1', size:1, used:0, free:1, devices:[{ rota:false }] }]), { status:200, headers:{'Content-Type':'application/json'} }))
  })

  it('saves updated options (applied)', async () => {
    vi.mocked(api.pools.getMountOptions).mockResolvedValue({ mountOptions:'compress=zstd:3,noatime' })
    vi.mocked(api.pools.updateMountOptions).mockResolvedValue({ ok:true, mountOptions:'compress=zstd:3,ssd,discard=async,noatime', rebootRequired:false })

    render(
      <MemoryRouter initialEntries={["/pools/p1"]}>
        <Routes>
          <Route path="/pools/:id" element={<PoolDetails />} />
        </Routes>
      </MemoryRouter>
    )

    await waitFor(() => expect(screen.getByText('Mount Options')).toBeInTheDocument())
    fireEvent.click(screen.getByText('Edit'))
    await waitFor(() => expect(screen.getByLabelText(/mount options/i)).toBeInTheDocument())
    const input = screen.getByLabelText(/mount options/i)
    fireEvent.change(input, { target: { value: 'compress=zstd:3,ssd,discard=async,noatime' } })
    fireEvent.click(screen.getByText('Save'))
    await waitFor(() => expect(toast.success).toHaveBeenCalledWith('Saved and applied.'))
  })

  it('saves with reboot warning', async () => {
    vi.mocked(api.pools.getMountOptions).mockResolvedValue({ mountOptions:'compress=zstd:3,noatime' })
    vi.mocked(api.pools.updateMountOptions).mockResolvedValue({ ok:true, mountOptions:'compress=zstd:3,ssd,discard=async,noatime', rebootRequired:true })

    render(
      <MemoryRouter initialEntries={["/pools/p1"]}>
        <Routes>
          <Route path="/pools/:id" element={<PoolDetails />} />
        </Routes>
      </MemoryRouter>
    )

    await waitFor(() => expect(screen.getByText('Mount Options')).toBeInTheDocument())
    fireEvent.click(screen.getByText('Edit'))
    await waitFor(() => expect(screen.getByLabelText(/mount options/i)).toBeInTheDocument())
    const input = screen.getByLabelText(/mount options/i)
    fireEvent.change(input, { target: { value: 'compress=zstd:3,ssd,discard=async,noatime' } })
    fireEvent.click(screen.getByText('Save'))
    await waitFor(() => expect(toast.success).toHaveBeenCalledWith('Saved. Will take effect after reboot or remount.'))
  })
})