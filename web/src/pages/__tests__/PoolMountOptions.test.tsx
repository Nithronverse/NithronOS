import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import * as toast from '@/components/ui/toast'
import * as api from '@/lib/api'
import { PoolDetails } from '../PoolDetails'

vi.mock('@/lib/api', async () => {
  const mod = await vi.importActual<any>('@/lib/api')
  return {
    ...mod,
    default: {
      ...mod.default,
      pools: {
        ...mod.default.pools,
        getMountOptions: vi.fn(),
        setMountOptions: vi.fn(),
      },
    },
  }
})

describe('PoolDetails Mount Options', () => {
  beforeEach(() => {
    vi.spyOn(window, 'fetch').mockResolvedValue(new Response(JSON.stringify([{ id:'p1', mount:'/mnt/p1', uuid:'U', raid:'raid1', size:1, used:0, free:1, devices:[{ rota:false }] }]), { status:200, headers:{'Content-Type':'application/json'} }))
  })

  it('saves updated options (applied)', async () => {
    const pushToast = vi.spyOn(toast, 'pushToast').mockImplementation(() => {})
    vi.mocked((api as any).default.pools.getMountOptions).mockResolvedValue({ mountOptions:'compress=zstd:3,noatime' })
    vi.mocked((api as any).default.pools.setMountOptions).mockResolvedValue({ ok:true, mountOptions:'compress=zstd:3,ssd,discard=async,noatime', rebootRequired:false })

    render(
      <MemoryRouter initialEntries={["/pools/p1"]}>
        <Routes>
          <Route path="/pools/:id" element={<PoolDetails />} />
        </Routes>
      </MemoryRouter>
    )

    await screen.findByText(/Pool Details/i)
    fireEvent.click(screen.getByRole('button', { name:/Edit/i }))
    const ta = await screen.findByRole('textbox') as HTMLTextAreaElement
    fireEvent.change(ta, { target: { value: 'compress=zstd:3,ssd,discard=async,noatime' } })
    fireEvent.click(screen.getByRole('button', { name:/Save/i }))
    await waitFor(() => expect(pushToast).toHaveBeenCalled())
  })

  it('saves updated options (requires reboot)', async () => {
    const pushToast = vi.spyOn(toast, 'pushToast').mockImplementation(() => {})
    vi.mocked((api as any).default.pools.getMountOptions).mockResolvedValue({ mountOptions:'compress=zstd:3,noatime' })
    vi.mocked((api as any).default.pools.setMountOptions).mockResolvedValue({ ok:true, mountOptions:'compress=zstd:3,noatime', rebootRequired:true })

    render(
      <MemoryRouter initialEntries={["/pools/p1"]}>
        <Routes>
          <Route path="/pools/:id" element={<PoolDetails />} />
        </Routes>
      </MemoryRouter>
    )
    await screen.findByText(/Pool Details/i)
    fireEvent.click(screen.getByRole('button', { name:/Edit/i }))
    const ta = await screen.findByRole('textbox') as HTMLTextAreaElement
    fireEvent.change(ta, { target: { value: 'compress=zstd:3,noatime' } })
    fireEvent.click(screen.getByRole('button', { name:/Save/i }))
    await waitFor(() => expect(pushToast).toHaveBeenCalled())
  })

  it('restore defaults uses SSD string when rota=0', async () => {
    vi.mocked((api as any).default.pools.getMountOptions).mockResolvedValue({ mountOptions:'compress=zstd:3,noatime' })
    render(
      <MemoryRouter initialEntries={["/pools/p1"]}>
        <Routes>
          <Route path="/pools/:id" element={<PoolDetails />} />
        </Routes>
      </MemoryRouter>
    )
    await screen.findByText(/Pool Details/i)
    fireEvent.click(screen.getByRole('button', { name:/Edit/i }))
    const ta = await screen.findByRole('textbox') as HTMLTextAreaElement
    fireEvent.click(screen.getByRole('button', { name:/Restore defaults/i }))
    expect(ta.value).toBe('compress=zstd:3,ssd,discard=async,noatime')
  })
})


