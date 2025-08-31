import http from '@/lib/nos-client';
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
  getOverview: () => http.get<MonitorOverview>('/v1/monitor/overview'),
  
  queryTimeSeries: (query: TimeSeriesQuery) => 
    http.post<TimeSeries>('/v1/monitor/timeseries', query),
  
  getDevices: () => http.get<{ disks: DiskMetrics[] }>('/v1/monitor/devices'),
  
  getServices: () => http.get<{ services: ServiceMetrics[] }>('/v1/monitor/services'),
  
  getBtrfs: () => http.get<{ btrfs: BtrfsMetrics[] }>('/v1/monitor/btrfs'),
  
  // Alert rules
  alerts: {
    rules: {
      list: () => http.get<{ rules: AlertRule[] }>('/v1/monitor/alerts/rules'),
      get: (id: string) => http.get<AlertRule>(`/v1/monitor/alerts/rules/${id}`),
      create: (rule: Partial<AlertRule>) => 
        http.post<AlertRule>('/v1/monitor/alerts/rules', rule),
      update: (id: string, rule: Partial<AlertRule>) => 
        http.patch<AlertRule>(`/v1/monitor/alerts/rules/${id}`, rule),
      delete: (id: string) => 
        http.del<{ status: string }>(`/v1/monitor/alerts/rules/${id}`),
    },
    
    // Notification channels
    channels: {
      list: () => http.get<{ channels: NotificationChannel[] }>('/v1/monitor/alerts/channels'),
      get: (id: string) => http.get<NotificationChannel>(`/v1/monitor/alerts/channels/${id}`),
      create: (channel: Partial<NotificationChannel>) => 
        http.post<NotificationChannel>('/v1/monitor/alerts/channels', channel),
      update: (id: string, channel: Partial<NotificationChannel>) => 
        http.patch<NotificationChannel>(`/v1/monitor/alerts/channels/${id}`, channel),
      delete: (id: string) => 
        http.del<{ status: string }>(`/v1/monitor/alerts/channels/${id}`),
      test: (id: string) => 
        http.post<{ success: boolean; error?: string }>(`/v1/monitor/alerts/channels/${id}/test`, {}),
    },
    
    // Events
    events: {
      list: (limit?: number) => {
        const params = limit ? `?limit=${limit}` : '';
        return http.get<{ events: AlertEvent[] }>(`/v1/monitor/alerts/events${params}`);
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
