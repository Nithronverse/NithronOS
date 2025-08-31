import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import SettingsSchedules from '../../routes/settings/schedules'
import { toast } from '@/components/ui/toast'

vi.mock('@/components/ui/toast', () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}))

vi.mock('@/lib/nos-client', () => {
  const get = vi.fn()
  const post = vi.fn()
  return { default: { get, post } }
})

import http from '@/lib/nos-client'

describe('SettingsSchedules', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(http.get).mockReset()
    vi.mocked(http.post).mockReset()
  })

  it('loads and populates fields', async () => {
    vi.mocked(http.get).mockResolvedValueOnce({ smartScan: 'Sun 03:00', btrfsScrub: 'Sun *-*-01..07 03:00', updatedAt: 'now' } as any)
    render(<MemoryRouter><SettingsSchedules /></MemoryRouter>)
    const smart = await screen.findByPlaceholderText('Sun 03:00') as HTMLInputElement
    const scrub = await screen.findByPlaceholderText('Sun *-*-01..07 03:00') as HTMLInputElement
    expect(smart.value).toBe('Sun 03:00')
    expect(scrub.value).toBe('Sun *-*-01..07 03:00')
    expect(await screen.findByText(/Examples: "Sun 03:00", "Wed 02:00"/i)).toBeTruthy()
    expect(await screen.findByText(/first Sunday monthly/i)).toBeTruthy()
  })

  it('client-validates required and format, focusing first invalid field', async () => {
    vi.mocked(http.get).mockResolvedValueOnce({ smartScan: 'Sun 03:00', btrfsScrub: 'Sun *-*-01..07 03:00' } as any)
    render(<MemoryRouter><SettingsSchedules /></MemoryRouter>)
    const smart = await screen.findByPlaceholderText('Sun 03:00') as HTMLInputElement
    const scrub = await screen.findByPlaceholderText('Sun *-*-01..07 03:00') as HTMLInputElement
    fireEvent.change(smart, { target: { value: '' } })
    fireEvent.change(scrub, { target: { value: 'NoSpacePattern' } })
    const save = screen.getByRole('button', { name: /save/i })
    fireEvent.click(save)
    expect(await screen.findByText(/Required/i)).toBeTruthy()
    expect(await screen.findByText(/Must include a space/i)).toBeTruthy()
    await waitFor(() => {
      expect((document.activeElement as HTMLInputElement)?.name).toBe('smartScan')
    })
  })

  it('shows backend 422 field hint and focuses that field', async () => {
    vi.mocked(http.get).mockResolvedValueOnce({ smartScan: 'Sun 03:00', btrfsScrub: 'Sun *-*-01..07 03:00' } as any)
    const err: any = new Error('HTTP 422')
    err.status = 422
    err.data = { error: { message:'invalid schedule', code: 'schedule.invalid', details: { field:'btrfsScrub', hint:'Use first Sunday monthly pattern' } } }
    vi.mocked(http.post).mockRejectedValueOnce(err)

    render(<MemoryRouter><SettingsSchedules /></MemoryRouter>)
    const smart = await screen.findByPlaceholderText('Sun 03:00') as HTMLInputElement
    const scrub = await screen.findByPlaceholderText('Sun *-*-01..07 03:00') as HTMLInputElement
    fireEvent.change(smart, { target: { value: 'Mon 03:00' } })
    fireEvent.change(scrub, { target: { value: 'Sun *-*-01..07 03:00' } })
    const save = screen.getByRole('button', { name: /save/i })
    fireEvent.click(save)
    expect(await screen.findByText(/HTTP 422/i)).toBeTruthy()
    expect(await screen.findByText(/Use first Sunday monthly pattern/i)).toBeTruthy()
    await waitFor(() => {
      expect((document.activeElement as HTMLInputElement)?.name).toBe('btrfsScrub')
    })
  })

  it('toasts on 5xx and re-enables Save after completion', async () => {
    vi.mocked(http.get).mockResolvedValueOnce({ smartScan: 'Sun 03:00', btrfsScrub: 'Sun *-*-01..07 03:00' } as any)
    const serverErr = new Error('server error') as any
    serverErr.status = 500
    vi.mocked(http.post).mockRejectedValueOnce(serverErr)
    const toastErrorSpy = vi.spyOn(toast, 'error')

    render(<MemoryRouter><SettingsSchedules /></MemoryRouter>)
    await screen.findByPlaceholderText('Sun 03:00')
    const save = screen.getByRole('button', { name: /save/i }) as HTMLButtonElement
    fireEvent.click(save)
    expect(save.disabled).toBe(true)
    await waitFor(() => expect(toastErrorSpy).toHaveBeenCalledWith('Failed to save schedules'))
    await waitFor(() => expect(save.disabled).toBe(false))
  })

  it('restore defaults fills fields without saving', async () => {
    vi.mocked(http.get).mockResolvedValueOnce({ smartScan: 'Mon 04:00', btrfsScrub: 'Mon *-*-01..07 04:00' } as any)
    const postSpy = vi.mocked(http.post)

    render(<MemoryRouter><SettingsSchedules /></MemoryRouter>)
    const smart = await screen.findByPlaceholderText('Sun 03:00') as HTMLInputElement
    const scrub = await screen.findByPlaceholderText('Sun *-*-01..07 03:00') as HTMLInputElement
    const restoreBtn = await screen.findByRole('button', { name: /restore defaults/i })
    fireEvent.click(restoreBtn)
    expect(smart.value).toBe('Sun 03:00')
    expect(scrub.value).toBe('Sun *-*-01..07 03:00')
    expect(postSpy).not.toHaveBeenCalled()
  })
})


