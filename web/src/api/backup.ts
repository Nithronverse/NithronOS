import http from '@/lib/nos-client';
import type {
  Schedule,
  Snapshot,
  Destination,
  BackupJob,
  RestorePlan,
  RestorePoint,
  SnapshotStats,
  CreateScheduleRequest,
  CreateSnapshotRequest,
  CreateDestinationRequest,
  ReplicateRequest,
  RestorePlanRequest,
  StoreSSHKeyRequest,
} from './backup.types';

export const backupApi = {
  // Schedules
  schedules: {
    list: () => http.get<{ schedules: Schedule[] }>('/v1/backup/schedules'),
    get: (id: string) => http.get<Schedule>(`/v1/backup/schedules/${id}`),
    create: (data: CreateScheduleRequest) => http.post<Schedule>('/v1/backup/schedules', data),
    update: (id: string, data: Partial<Schedule>) => http.patch<Schedule>(`/v1/backup/schedules/${id}`, data),
    delete: (id: string) => http.del<{ status: string }>(`/v1/backup/schedules/${id}`),
  },
  
  // Snapshots
  snapshots: {
    list: (subvolume?: string) => {
      const params = subvolume ? `?subvolume=${encodeURIComponent(subvolume)}` : '';
      return http.get<{ snapshots: Snapshot[] }>(`/v1/backup/snapshots${params}`);
    },
    create: (data: CreateSnapshotRequest) => http.post<BackupJob>('/v1/backup/snapshots/create', data),
    delete: (id: string) => http.del<{ status: string }>(`/v1/backup/snapshots/${id}`),
    stats: () => http.get<SnapshotStats>('/v1/backup/snapshots/stats'),
  },
  
  // Destinations
  destinations: {
    list: () => http.get<{ destinations: Destination[] }>('/v1/backup/destinations'),
    get: (id: string) => http.get<Destination>(`/v1/backup/destinations/${id}`),
    create: (data: CreateDestinationRequest) => http.post<Destination>('/v1/backup/destinations', data),
    update: (id: string, data: Partial<Destination>) => http.patch<Destination>(`/v1/backup/destinations/${id}`, data),
    delete: (id: string) => http.del<{ status: string }>(`/v1/backup/destinations/${id}`),
    test: (id: string) => http.post<{ success: boolean; error?: string }>(`/v1/backup/destinations/${id}/test`, {}),
    storeSSHKey: (id: string, data: StoreSSHKeyRequest) => http.post<{ status: string }>(`/v1/backup/destinations/${id}/key`, data),
  },
  
  // Replication
  replicate: (data: ReplicateRequest) => http.post<BackupJob>('/v1/backup/replicate', data),
  
  // Restore
  restore: {
    createPlan: (data: RestorePlanRequest) => http.post<RestorePlan>('/v1/backup/restore/plan', data),
    apply: (data: RestorePlanRequest) => http.post<BackupJob>('/v1/backup/restore/apply', data),
    listPoints: () => http.get<{ restore_points: RestorePoint[] }>('/v1/backup/restore/points'),
  },
  
  // Jobs
  jobs: {
    list: (limit?: number) => {
      const params = limit ? `?limit=${limit}` : '';
      return http.get<{ jobs: BackupJob[] }>(`/v1/backup/jobs${params}`);
    },
    get: (id: string) => http.get<BackupJob>(`/v1/backup/jobs/${id}`),
    cancel: (id: string) => http.post<{ status: string }>(`/v1/backup/jobs/${id}/cancel`, {}),
  },
};

// Export types for convenience
export type {
  Schedule,
  Snapshot,
  Destination,
  BackupJob,
  RestorePlan,
  RestorePoint,
  SnapshotStats,
} from './backup.types';
