import { get, post, patch, del } from '@/lib/api';
import type {
  MonitorOverview,
  TimeSeries,
  TimeSeriesQuery,
  DiskMetrics,
  ServiceMetrics,
  BtrfsMetrics,
  AlertRule,
  NotificationChannel,
  AlertEvent,
} from './monitor.types';

export const monitorApi = {
  // Metrics
  getOverview: () => get<MonitorOverview>('/api/v1/monitor/overview'),
  
  queryTimeSeries: (query: TimeSeriesQuery) => 
    post<TimeSeries>('/api/v1/monitor/timeseries', query),
  
  getDevices: () => get<{ disks: DiskMetrics[] }>('/api/v1/monitor/devices'),
  
  getServices: () => get<{ services: ServiceMetrics[] }>('/api/v1/monitor/services'),
  
  getBtrfs: () => get<{ btrfs: BtrfsMetrics[] }>('/api/v1/monitor/btrfs'),
  
  // Alert rules
  alerts: {
    rules: {
      list: () => get<{ rules: AlertRule[] }>('/api/v1/monitor/alerts/rules'),
      get: (id: string) => get<AlertRule>(`/api/v1/monitor/alerts/rules/${id}`),
      create: (rule: Partial<AlertRule>) => 
        post<AlertRule>('/api/v1/monitor/alerts/rules', rule),
      update: (id: string, rule: Partial<AlertRule>) => 
        patch<AlertRule>(`/api/v1/monitor/alerts/rules/${id}`, rule),
      delete: (id: string) => 
        del<{ status: string }>(`/api/v1/monitor/alerts/rules/${id}`),
    },
    
    // Notification channels
    channels: {
      list: () => get<{ channels: NotificationChannel[] }>('/api/v1/monitor/alerts/channels'),
      get: (id: string) => get<NotificationChannel>(`/api/v1/monitor/alerts/channels/${id}`),
      create: (channel: Partial<NotificationChannel>) => 
        post<NotificationChannel>('/api/v1/monitor/alerts/channels', channel),
      update: (id: string, channel: Partial<NotificationChannel>) => 
        patch<NotificationChannel>(`/api/v1/monitor/alerts/channels/${id}`, channel),
      delete: (id: string) => 
        del<{ status: string }>(`/api/v1/monitor/alerts/channels/${id}`),
      test: (id: string) => 
        post<{ success: boolean; error?: string }>(`/api/v1/monitor/alerts/channels/${id}/test`, {}),
    },
    
    // Events
    events: {
      list: (limit?: number) => {
        const params = limit ? `?limit=${limit}` : '';
        return get<{ events: AlertEvent[] }>(`/api/v1/monitor/alerts/events${params}`);
      },
    },
  },
};

// Export types for convenience
export type {
  MonitorOverview,
  SystemMetrics,
  CPUMetrics,
  MemoryMetrics,
  DiskMetrics,
  NetworkMetrics,
  ServiceMetrics,
  BtrfsMetrics,
  TimeSeries,
  AlertRule,
  NotificationChannel,
  AlertEvent,
} from './monitor.types';
