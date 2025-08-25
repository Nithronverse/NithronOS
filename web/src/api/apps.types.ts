// App Catalog Types

export interface CatalogEntry {
  id: string;
  name: string;
  version: string;
  description: string;
  categories: string[];
  icon: string;
  compose: string;
  schema: string;
  defaults: AppDefaults;
  health: HealthConfig;
  needs_privileged: boolean;
  notes?: string;
}

export interface AppDefaults {
  env: Record<string, string>;
  volumes: VolumeMount[];
  ports: PortMapping[];
  resources: ResourceLimits;
}

export interface VolumeMount {
  host: string;
  container: string;
  read_only?: boolean;
}

export interface PortMapping {
  host: number;
  container: number;
  protocol: string;
}

export interface ResourceLimits {
  cpu_limit?: string;
  memory_limit?: string;
  cpu_request?: string;
  mem_request?: string;
}

export interface HealthConfig {
  type: 'container' | 'http';
  container?: string;
  url?: string;
  interval_s: number;
  timeout_s: number;
  healthy_after: number;
  unhealthy_after: number;
}

export interface Catalog {
  version: string;
  entries: CatalogEntry[];
  source?: string;
  updated_at: string;
}

export interface InstalledApp {
  id: string;
  name: string;
  version: string;
  status: AppStatus;
  params: Record<string, any>;
  ports: PortMapping[];
  urls: string[];
  health: HealthStatus;
  installed_at: string;
  updated_at: string;
  snapshots: AppSnapshot[];
}

export type AppStatus = 
  | 'stopped'
  | 'starting'
  | 'running'
  | 'stopping'
  | 'error'
  | 'upgrading'
  | 'rollback'
  | 'unknown';

export interface HealthStatus {
  status: 'healthy' | 'unhealthy' | 'unknown';
  checked_at: string;
  message?: string;
  containers?: ContainerHealth[];
}

export interface ContainerHealth {
  name: string;
  status: string;
  health?: string;
}

export interface AppSnapshot {
  id: string;
  timestamp: string;
  type: 'btrfs' | 'rsync';
  name: string;
  path: string;
}

export interface InstallRequest {
  id: string;
  version?: string;
  params?: Record<string, any>;
}

export interface UpgradeRequest {
  version: string;
  params?: Record<string, any>;
}

export interface RollbackRequest {
  snapshot_ts: string;
}

export interface DeleteRequest {
  keep_data?: boolean;
}

export interface LogStreamOptions {
  follow?: boolean;
  tail?: number;
  timestamps?: boolean;
  container?: string;
}

export interface AppEvent {
  id: string;
  type: string;
  app_id: string;
  timestamp: string;
  user?: string;
  request_id?: string;
  details?: any;
}

export interface JsonSchema {
  type: string;
  title?: string;
  description?: string;
  properties?: Record<string, JsonSchemaProperty>;
  required?: string[];
}

export interface JsonSchemaProperty {
  type: string;
  title?: string;
  description?: string;
  default?: any;
  enum?: any[];
  format?: string;
  pattern?: string;
  minLength?: number;
  maxLength?: number;
  minimum?: number;
  maximum?: number;
  examples?: any[];
}
