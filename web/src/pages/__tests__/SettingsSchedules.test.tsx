import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import SettingsSchedules from '../../routes/settings/schedules'
import * as toast from '@/components/ui/toast'

function installFetchMock(handler: (input: RequestInfo, init?: RequestInit) => Promise<Response>) {
  const original = global.fetch
  // @ts-ignore
  global.fetch = vi.fn(handler) as unknown as typeof fetch
  return () => { global.fetch = original }
}

function schedulesGetResponse(data: { smartScan: string; btrfsScrub: string; updatedAt?: string }) {
  return new Response(JSON.stringify(data), { status:200, headers:{ 'Content-Type':'application/json' } })
}

describe('SettingsSchedules', () => {
  let restore: (()=>void) | undefined

  beforeEach(() => {
    restore?.()
    restore = installFetchMock(async (input, init) => {
      const url = typeof input === 'string' ? input : input.toString()
      const method = (init?.method || 'GET').toUpperCase()
      if (url.includes('/api/v1/schedules') && method === 'GET') {
        return schedulesGetResponse({ smartScan: 'Sun 03:00', btrfsScrub: 'Sun *-*-01..07 03:00', updatedAt: 'now' })
      }
      return new Response('{}', { status:200, headers:{ 'Content-Type':'application/json' } })
    })
  })

  it('loads and populates fields', async () => {
    render(<MemoryRouter><SettingsSchedules /></MemoryRouter>)
    const smart = await screen.findByPlaceholderText('Sun 03:00') as HTMLInputElement
    const scrub = await screen.findByPlaceholderText('Sun *-*-01..07 03:00') as HTMLInputElement
    expect(smart.value).toBe('Sun 03:00')
    expect(scrub.value).toBe('Sun *-*-01..07 03:00')
    // helper texts present
    expect(await screen.findByText(/Examples: "Sun 03:00", "Wed 02:00"/i)).toBeTruthy()
    expect(await screen.findByText(/first Sunday monthly/i)).toBeTruthy()
  })

  it('client-validates required and format, focusing first invalid field', async () => {
    render(<MemoryRouter><SettingsSchedules /></MemoryRouter>)
    const smart = await screen.findByPlaceholderText('Sun 03:00') as HTMLInputElement
    const scrub = await screen.findByPlaceholderText('Sun *-*-01..07 03:00') as HTMLInputElement
    // Make invalid values
    fireEvent.change(smart, { target: { value: '' } })
    fireEvent.change(scrub, { target: { value: 'NoSpacePattern' } })
    const save = screen.getByRole('button', { name: /save/i })
    fireEvent.click(save)
    // errors shown
    expect(await screen.findByText(/Required/i)).toBeTruthy()
    expect(await screen.findByText(/Must include a space/i)).toBeTruthy()
    // focus first invalid (smartScan)
    await waitFor(() => {
      expect((document.activeElement as HTMLInputElement)?.name).toBe('smartScan')
    })
  })

  it('shows backend 422 field hint and focuses that field', async () => {
    restore?.()
    restore = installFetchMock(async (input, init) => {
      const url = typeof input === 'string' ? input : input.toString()
      const method = (init?.method || 'GET').toUpperCase()
      if (url.includes('/api/v1/schedules') && method === 'GET') {
        return schedulesGetResponse({ smartScan: 'Sun 03:00', btrfsScrub: 'Sun *-*-01..07 03:00' })
      }
      if (url.includes('/api/v1/schedules') && method === 'POST') {
        const body = { error: { message:'invalid schedule', code: 'schedule.invalid', details: { field:'btrfsScrub', hint:'Use first Sunday monthly pattern' } } }
        return new Response(JSON.stringify(body), { status:422, headers:{ 'Content-Type':'application/json' } })
      }
      return new Response('{}', { status:200, headers:{ 'Content-Type':'application/json' } })
    })

    render(<MemoryRouter><SettingsSchedules /></MemoryRouter>)
    const smart = await screen.findByPlaceholderText('Sun 03:00') as HTMLInputElement
    const scrub = await screen.findByPlaceholderText('Sun *-*-01..07 03:00') as HTMLInputElement
    // Values remain valid to pass client validation
    fireEvent.change(smart, { target: { value: 'Mon 03:00' } })
    fireEvent.change(scrub, { target: { value: 'Sun *-*-01..07 03:00' } })
    const save = screen.getByRole('button', { name: /save/i })
    fireEvent.click(save)
    // Inline generic error and field-specific hint
    expect(await screen.findByText(/HTTP 422/i)).toBeTruthy()
    expect(await screen.findByText(/Use first Sunday monthly pattern/i)).toBeTruthy()
    await waitFor(() => {
      expect((document.activeElement as HTMLInputElement)?.name).toBe('btrfsScrub')
    })
  })

  it('toasts on 5xx and re-enables Save after completion', async () => {
    restore?.()
    restore = installFetchMock(async (input, init) => {
      const url = typeof input === 'string' ? input : input.toString()
      const method = (init?.method || 'GET').toUpperCase()
      if (url.includes('/api/v1/schedules') && method === 'GET') {
        return schedulesGetResponse({ smartScan: 'Sun 03:00', btrfsScrub: 'Sun *-*-01..07 03:00' })
      }
      if (url.includes('/api/v1/schedules') && method === 'POST') {
        return new Response(JSON.stringify({ error: { message:'server error' } }), { status:500, headers:{ 'Content-Type':'application/json' } })
      }
      return new Response('{}', { status:200, headers:{ 'Content-Type':'application/json' } })
    })
    const toastSpy = vi.spyOn(toast, 'pushToast')
    render(<MemoryRouter><SettingsSchedules /></MemoryRouter>)
    await screen.findByPlaceholderText('Sun 03:00')
    const save = screen.getByRole('button', { name: /save/i }) as HTMLButtonElement
    fireEvent.click(save)
    expect(save.disabled).toBe(true)
    await waitFor(() => expect(toastSpy).toHaveBeenCalled())
    await waitFor(() => expect(save.disabled).toBe(false))
  })

  it('restore defaults fills fields without saving', async () => {
    const original = global.fetch
    const postSpy = vi.fn()
    // @ts-ignore
    global.fetch = vi.fn(async (input: RequestInfo, init?: RequestInit) => {
      const url = typeof input === 'string' ? input : input.toString()
      const method = (init?.method || 'GET').toUpperCase()
      if (url.includes('/api/v1/schedules') && method === 'GET') {
        return schedulesGetResponse({ smartScan: 'Mon 04:00', btrfsScrub: 'Mon *-*-01..07 04:00' })
      }
      if (url.includes('/api/v1/schedules') && method === 'POST') {
        postSpy()
        return new Response('{}', { status:200, headers:{ 'Content-Type':'application/json' } })
      }
      return new Response('{}', { status:200, headers:{ 'Content-Type':'application/json' } })
    }) as unknown as typeof fetch

    render(<MemoryRouter><SettingsSchedules /></MemoryRouter>)
    const smart = await screen.findByPlaceholderText('Sun 03:00') as HTMLInputElement
    const scrub = await screen.findByPlaceholderText('Sun *-*-01..07 03:00') as HTMLInputElement
    const restoreBtn = await screen.findByRole('button', { name: /restore defaults/i })
    fireEvent.click(restoreBtn)
    expect(smart.value).toBe('Sun 03:00')
    expect(scrub.value).toBe('Sun *-*-01..07 03:00')
    // Ensure no save network call happened automatically
    expect(postSpy).not.toHaveBeenCalled()
    global.fetch = original
  })
})


