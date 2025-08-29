import { get, post, patch, del } from '@/lib/api';
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
    list: () => get<{ schedules: Schedule[] }>('/api/v1/backup/schedules'),
    get: (id: string) => get<Schedule>(`/api/v1/backup/schedules/${id}`),
    create: (data: CreateScheduleRequest) => post<Schedule>('/api/v1/backup/schedules', data),
    update: (id: string, data: Partial<Schedule>) => patch<Schedule>(`/api/v1/backup/schedules/${id}`, data),
    delete: (id: string) => del<{ status: string }>(`/api/v1/backup/schedules/${id}`),
  },
  
  // Snapshots
  snapshots: {
    list: (subvolume?: string) => {
      const params = subvolume ? `?subvolume=${encodeURIComponent(subvolume)}` : '';
      return get<{ snapshots: Snapshot[] }>(`/api/v1/backup/snapshots${params}`);
    },
    create: (data: CreateSnapshotRequest) => post<BackupJob>('/api/v1/backup/snapshots/create', data),
    delete: (id: string) => del<{ status: string }>(`/api/v1/backup/snapshots/${id}`),
    stats: () => get<SnapshotStats>('/api/v1/backup/snapshots/stats'),
  },
  
  // Destinations
  destinations: {
    list: () => get<{ destinations: Destination[] }>('/api/v1/backup/destinations'),
    get: (id: string) => get<Destination>(`/api/v1/backup/destinations/${id}`),
    create: (data: CreateDestinationRequest) => post<Destination>('/api/v1/backup/destinations', data),
    update: (id: string, data: Partial<Destination>) => patch<Destination>(`/api/v1/backup/destinations/${id}`, data),
    delete: (id: string) => del<{ status: string }>(`/api/v1/backup/destinations/${id}`),
    test: (id: string) => post<{ success: boolean; error?: string }>(`/api/v1/backup/destinations/${id}/test`, {}),
    storeSSHKey: (id: string, data: StoreSSHKeyRequest) => post<{ status: string }>(`/api/v1/backup/destinations/${id}/key`, data),
  },
  
  // Replication
  replicate: (data: ReplicateRequest) => post<BackupJob>('/api/v1/backup/replicate', data),
  
  // Restore
  restore: {
    createPlan: (data: RestorePlanRequest) => post<RestorePlan>('/api/v1/backup/restore/plan', data),
    apply: (data: RestorePlanRequest) => post<BackupJob>('/api/v1/backup/restore/apply', data),
    listPoints: () => get<{ restore_points: RestorePoint[] }>('/api/v1/backup/restore/points'),
  },
  
  // Jobs
  jobs: {
    list: (limit?: number) => {
      const params = limit ? `?limit=${limit}` : '';
      return get<{ jobs: BackupJob[] }>(`/api/v1/backup/jobs${params}`);
    },
    get: (id: string) => get<BackupJob>(`/api/v1/backup/jobs/${id}`),
    cancel: (id: string) => post<{ status: string }>(`/api/v1/backup/jobs/${id}/cancel`, {}),
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
