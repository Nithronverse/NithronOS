import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { api } from '@/lib/api-client'
import { PoolDetails } from '../PoolDetails'

vi.mock('@/lib/api-client', () => ({
  api: {
    pools: {
      list: vi.fn(),
      get: vi.fn(),
      planDevice: vi.fn(),
      applyDevice: vi.fn(),
      getMountOptions: vi.fn().mockResolvedValue({ mountOptions: 'compress=zstd:3,noatime' }),
    },
  },
}))

describe('Devices wizard warnings and confirm gating', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    // Mock the pool data
    vi.mocked(api.pools.get).mockResolvedValue({
      id:'p1', 
      mount:'/mnt/p1', 
      uuid:'U', 
      raid:'raid1', 
      size:1, 
      used:0, 
      free:1, 
      devices:[]
    })
  })

  it('shows plan warnings and blocks Apply until confirm matches', async () => {
    vi.mocked(api.pools.planDevice).mockResolvedValue({
      planId: 'x',
      steps: [{ id:'s1', description:'add devices', command:"btrfs device add /dev/sdb /mnt/p1" }],
      warnings: ['Pool is >80% full; balance may take longer.'],
      requiresBalance: true,
    })
    vi.mocked(api.pools.applyDevice).mockResolvedValue({ ok:true, tx_id:'t1' })

    render(
      <MemoryRouter initialEntries={["/pools/p1"]}>
        <Routes>
          <Route path="/pools/:id" element={<PoolDetails />} />
        </Routes>
      </MemoryRouter>
    )

    await screen.findByText(/Pool Details/i)
    fireEvent.click(screen.getByRole('button', { name:/Devices/i }))
    fireEvent.click(screen.getByRole('button', { name:/Add/i }))
    fireEvent.change(screen.getByLabelText(/Devices to add/i), { target: { value: '/dev/sdb' } })
    fireEvent.click(screen.getByRole('button', { name:/Plan/i }))

    await screen.findByText(/Warnings:/i)
    const applyBtn = screen.getByRole('button', { name:/Apply/i }) as HTMLButtonElement
    expect(applyBtn.disabled).toBe(true)

    const confirmInput = screen.getByLabelText(/Confirm code/i)
    fireEvent.change(confirmInput, { target: { value: 'ADD' } })

    await waitFor(() => expect(applyBtn.disabled).toBe(false))

    fireEvent.click(applyBtn)
    await waitFor(() => expect(api.pools.applyDevice).toHaveBeenCalled())
  })
})


