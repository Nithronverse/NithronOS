import http from '@/lib/nos-client'

export type Schedules = { smartScan: string; btrfsScrub: string; updatedAt?: string }
export type SchedulesUpdate = { smartScan: string; btrfsScrub: string }

export async function getSchedules(): Promise<Schedules> {
  return http.get<Schedules>('/v1/schedules')
}

export async function updateSchedules(input: SchedulesUpdate): Promise<Schedules> {
  return http.post<Schedules>('/v1/schedules', input)
}


