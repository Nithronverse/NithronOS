import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import SettingsSchedules from '../schedules'

vi.mock('@/api/schedules', () => ({
  getSchedules: vi.fn(),
  updateSchedules: vi.fn(),
}))

vi.mock('@/components/ui/toast', () => ({
  pushToast: vi.fn(),
}))

import { getSchedules, updateSchedules } from '@/api/schedules'
import { pushToast } from '@/components/ui/toast'

describe('SettingsSchedules (route)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    ;(getSchedules as any).mockResolvedValue({ smartScan: 'Sun 03:00', btrfsScrub: 'Sun *-*-01..07 03:00', updatedAt: 'now' })
  })

  it('saves updated schedules and toasts success', async () => {
    ;(updateSchedules as any).mockImplementation(async (input: any) => ({ ...input, updatedAt: 'later' }))

    render(
      <MemoryRouter>
        <SettingsSchedules />
      </MemoryRouter>,
    )

    const smart = await screen.findByPlaceholderText('Sun 03:00') as HTMLInputElement
    const scrub = await screen.findByPlaceholderText('Sun *-*-01..07 03:00') as HTMLInputElement

    fireEvent.change(smart, { target: { value: 'Tue 01:23' } })
    fireEvent.change(scrub, { target: { value: 'Sun *-*-01..07 05:00' } })

    fireEvent.click(screen.getByRole('button', { name: /save/i }))

    await waitFor(() => {
      expect(updateSchedules).toHaveBeenCalledWith({ smartScan: 'Tue 01:23', btrfsScrub: 'Sun *-*-01..07 05:00' })
      expect(pushToast).toHaveBeenCalled()
    })
  })

  it('renders inline hint on 422 schedule.invalid for btrfsScrub', async () => {
    const err: any = new Error('HTTP 422: invalid schedule')
    err.status = 422
    err.data = { error: { code: 'schedule.invalid', details: { field: 'btrfsScrub', hint: 'Use first Sunday monthly pattern' } } }
    ;(updateSchedules as any).mockRejectedValue(err)

    render(
      <MemoryRouter>
        <SettingsSchedules />
      </MemoryRouter>,
    )

    // Wait for initial load
    await screen.findByPlaceholderText('Sun 03:00')

    fireEvent.click(screen.getByRole('button', { name: /save/i }))

    expect(await screen.findByText(/Use first Sunday monthly pattern/i)).toBeTruthy()
  })
})


