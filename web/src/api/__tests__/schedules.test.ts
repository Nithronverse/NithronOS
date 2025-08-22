import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { getSchedules, updateSchedules, type Schedules, type SchedulesUpdate } from '../schedules'

const g: any = globalThis

describe('schedules api', () => {
  const originalFetch = g.fetch
  beforeEach(() => { g.fetch = vi.fn() })
  afterEach(() => { g.fetch = originalFetch })

  it('gets schedules', async () => {
    const payload: Schedules = { smartScan: 'Sun 03:00', btrfsScrub: 'Sun *-**-01..07 03:00', updatedAt: 'now' }
    g.fetch.mockResolvedValue({ ok: true, status: 200, json: async () => payload, headers: new Headers() })
    const res = await getSchedules()
    expect(res.smartScan).toBe(payload.smartScan)
    expect(res.btrfsScrub).toBe(payload.btrfsScrub)
  })

  it('handles validation error 422', async () => {
    const errBody = { error: { message: 'invalid schedule' } }
    g.fetch.mockResolvedValue({ ok: false, status: 422, json: async () => errBody, headers: new Headers({ 'content-type': 'application/json' }) })
    const input: SchedulesUpdate = { smartScan: '', btrfsScrub: '' }
    await expect(updateSchedules(input)).rejects.toThrow(/422/i)
  })
})


