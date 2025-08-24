import { describe, it, expect, vi } from 'vitest'
import { ErrProxyMisconfigured, fetchJSON } from '../http'

describe('fetchJSON', () => {
  it('throws ErrProxyMisconfigured when content-type is text/html', async () => {
    // @ts-ignore
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      headers: { get: () => 'text/html; charset=utf-8' },
      text: async () => '<!doctype html><html>oops</html>',
    })
    await expect(fetchJSON('/api/setup/state')).rejects.toBeInstanceOf(ErrProxyMisconfigured)
  })
})


