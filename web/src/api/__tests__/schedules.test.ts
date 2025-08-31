import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { getSchedules, updateSchedules, type Schedules, type SchedulesUpdate } from '../schedules'

vi.mock('@/lib/nos-client', () => {
  const get = vi.fn()
  const post = vi.fn()
  return { default: { get, post } }
})

import http from '@/lib/nos-client'

describe('schedules api', () => {
  beforeEach(() => { vi.clearAllMocks() })
  afterEach(() => {})

  it('gets schedules', async () => {
    const payload: Schedules = { smartScan: 'Sun 03:00', btrfsScrub: 'Sun *-**-01..07 03:00', updatedAt: 'now' }
    vi.mocked(http.get).mockResolvedValueOnce(payload as any)
    const res = await getSchedules()
    expect(http.get).toHaveBeenCalledWith('/v1/schedules')
    expect(res.smartScan).toBe(payload.smartScan)
    expect(res.btrfsScrub).toBe(payload.btrfsScrub)
  })

  it('handles validation error 422', async () => {
    const err: any = new Error('HTTP 422')
    err.status = 422
    vi.mocked(http.post).mockRejectedValueOnce(err)
    const input: SchedulesUpdate = { smartScan: '', btrfsScrub: '' }
    await expect(updateSchedules(input)).rejects.toThrow(/422/i)
  })
})


