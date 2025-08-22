import { apiGet, apiPost } from './http'

export type Schedules = { smartScan: string; btrfsScrub: string; updatedAt?: string }
export type SchedulesUpdate = { smartScan: string; btrfsScrub: string }

export async function getSchedules(): Promise<Schedules> {
  return apiGet<Schedules>('/api/v1/schedules')
}

export async function updateSchedules(input: SchedulesUpdate): Promise<Schedules> {
  return apiPost<Schedules>('/api/v1/schedules', input)
}


