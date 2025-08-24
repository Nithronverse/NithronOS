import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import App from '../App'

describe('Global banner on proxy misconfig', () => {
  it('shows banner when /api/setup/state returns HTML', async () => {
    // HTML response for setup state
    // @ts-ignore
    global.fetch = vi.fn().mockImplementation((url: string) => {
      if (url === '/api/setup/state') {
        return Promise.resolve({
          ok: true,
          status: 200,
          headers: { get: () => 'text/html' },
          text: async () => '<!doctype html>nope',
        })
      }
      // default auth call
      return Promise.resolve({ ok: false, status: 401, headers: { get: () => 'application/json' }, json: async () => ({}) })
    })

    render(<App />)
    const el = await screen.findByText(/Backend unreachable or proxy misconfigured/i)
    expect(!!el).toBe(true)
    const link = await screen.findByRole('link', { name: /Troubleshooting/i })
    expect(link.getAttribute('href')).toBe('/help/proxy')
  })
})


