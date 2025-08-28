import { apiClient } from '@/lib/api-client';
import type {
  Catalog,
  CatalogEntry,
  InstalledApp,
  InstallRequest,
  UpgradeRequest,
  RollbackRequest,
  LogStreamOptions,
  AppEvent,
} from './apps.types';

export const appsApi = {
  // Catalog operations
  getCatalog: () =>
    apiClient.get<Catalog>('/api/v1/apps/catalog'),

  getInstalledApps: () =>
    apiClient.get<{ items: InstalledApp[] }>('/api/v1/apps/installed'),

  getApp: (id: string) =>
    apiClient.get<InstalledApp>(`/api/v1/apps/${id}`),

  // App lifecycle operations
  installApp: (data: InstallRequest) =>
    apiClient.post<{ message: string; app: InstalledApp }>('/api/v1/apps/install', data),

  upgradeApp: (id: string, data: UpgradeRequest) =>
    apiClient.post<{ message: string; version: string }>(`/api/v1/apps/${id}/upgrade`, data),

  startApp: (id: string) =>
    apiClient.post<{ message: string }>(`/api/v1/apps/${id}/start`),

  stopApp: (id: string) =>
    apiClient.post<{ message: string }>(`/api/v1/apps/${id}/stop`),

  restartApp: (id: string) =>
    apiClient.post<{ message: string }>(`/api/v1/apps/${id}/restart`),

  deleteApp: (id: string, keepData = false) =>
    apiClient.delete<{ message: string }>(`/api/v1/apps/${id}?keep_data=${keepData}`),

  rollbackApp: (id: string, snapshotTs: string) =>
    apiClient.post<{ message: string }>(`/api/v1/apps/${id}/rollback`, {
      snapshot_ts: snapshotTs,
    } as RollbackRequest),

  // Monitoring operations
  forceHealthCheck: (id: string) =>
    apiClient.post<{ message: string; health: any }>(`/api/v1/apps/${id}/health`),

  getAppEvents: (id: string, limit = 100) =>
    apiClient.get<{ events: AppEvent[] }>(`/api/v1/apps/${id}/events?limit=${limit}`),

  // Admin operations
  syncCatalogs: () =>
    apiClient.post<{ message: string }>('/api/v1/apps/catalog/sync'),

  // WebSocket for logs
  streamLogs: (id: string, options: LogStreamOptions = {}) => {
    const params = new URLSearchParams();
    if (options.follow) params.append('follow', '1');
    if (options.tail) params.append('tail', options.tail.toString());
    if (options.timestamps) params.append('timestamps', 'true');
    if (options.container) params.append('container', options.container);

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host;
    const url = `${protocol}//${host}/api/v1/apps/${id}/logs?${params}`;
    
    return new WebSocket(url);
  },

  // Get logs (non-streaming)
  getLogs: (id: string, options: LogStreamOptions = {}) => {
    const params = new URLSearchParams();
    if (options.tail) params.append('tail', options.tail.toString());
    if (options.timestamps) params.append('timestamps', 'true');
    if (options.container) params.append('container', options.container);

    return apiClient.get<string>(`/api/v1/apps/${id}/logs?${params}`, {
      responseType: 'text',
    });
  },
};

// Helper to load schema from catalog entry
export async function loadAppSchema(catalogEntry: CatalogEntry): Promise<any> {
  if (!catalogEntry.schema) return null;
  
  // In production, schemas would be loaded from the server
  // For now, we'll parse them from the catalog
  try {
    const response = await fetch(`/api/v1/apps/schema/${catalogEntry.id}`);
    if (response.ok) {
      return await response.json();
    }
  } catch (error) {
    console.error('Failed to load schema:', error);
  }
  
  return null;
}
