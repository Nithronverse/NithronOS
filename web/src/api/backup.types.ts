// Backup and Restore API Types

export interface Schedule {
  id: string;
  name: string;
  enabled: boolean;
  subvolumes: string[];
  frequency: ScheduleFrequency;
  retention: RetentionPolicy;
  pre_hooks?: string[];
  post_hooks?: string[];
  last_run?: string;
  next_run?: string;
  created_at: string;
  updated_at: string;
}

export interface ScheduleFrequency {
  type: 'cron' | 'hourly' | 'daily' | 'weekly' | 'monthly';
  cron?: string;
  hour?: number;
  minute?: number;
  day?: number;
  weekday?: number;
}

export interface RetentionPolicy {
  min_keep: number;
  days: number;
  weeks: number;
  months: number;
  years: number;
}

export interface Snapshot {
  id: string;
  subvolume: string;
  path: string;
  created_at: string;
  size_bytes?: number;
  schedule_id?: string;
  tags?: string[];
  read_only: boolean;
  parent?: string;
}

export interface Destination {
  id: string;
  name: string;
  type: 'ssh' | 'rclone' | 'local';
  enabled: boolean;
  
  // SSH specific
  host?: string;
  port?: number;
  user?: string;
  path?: string;
  key_ref?: string;
  
  // Rclone specific
  remote_name?: string;
  remote_path?: string;
  
  // Common options
  bandwidth_limit?: number;
  concurrency?: number;
  retry_count?: number;
  
  last_test?: string;
  last_test_status?: string;
  created_at: string;
  updated_at: string;
}

export interface BackupJob {
  id: string;
  type: 'snapshot' | 'replicate' | 'restore';
  state: JobState;
  progress: number;
  
  // For snapshot jobs
  schedule_id?: string;
  subvolumes?: string[];
  
  // For replication jobs
  destination_id?: string;
  snapshot_id?: string;
  incremental?: boolean;
  base_snapshot?: string;
  
  // For restore jobs
  source_type?: string;
  restore_type?: string;
  restore_path?: string;
  
  // Common fields
  started_at: string;
  finished_at?: string;
  error?: string;
  bytes_total?: number;
  bytes_done?: number;
  
  // Logs
  log_entries?: LogEntry[];
}

export type JobState = 'pending' | 'running' | 'succeeded' | 'failed' | 'canceled';

export interface LogEntry {
  timestamp: string;
  level: 'info' | 'warn' | 'error';
  message: string;
}

export interface RestorePlan {
  source_type: string;
  source_id: string;
  restore_type: string;
  target_path: string;
  requires_stop?: string[];
  estimated_time_seconds: number;
  dry_run: boolean;
  actions: RestoreAction[];
}

export interface RestoreAction {
  type: 'stop_service' | 'snapshot' | 'copy' | 'rollback' | 'start_service' | 'mount' | 'unmount';
  target: string;
  description: string;
}

export interface RestorePoint {
  id: string;
  type: 'local' | 'ssh' | 'rclone';
  subvolume: string;
  timestamp: string;
  source: string;
  path: string;
}

export interface SnapshotStats {
  total_count: number;
  total_size_bytes: number;
  by_subvolume: Record<string, SubvolumeStats>;
  oldest_snapshot?: string;
  newest_snapshot?: string;
}

export interface SubvolumeStats {
  count: number;
  size_bytes: number;
  last_backup?: string;
}

// API Request/Response types

export interface CreateScheduleRequest {
  name: string;
  enabled: boolean;
  subvolumes: string[];
  frequency: ScheduleFrequency;
  retention: RetentionPolicy;
  pre_hooks?: string[];
  post_hooks?: string[];
}

export interface CreateSnapshotRequest {
  subvolumes: string[];
  tag?: string;
}

export interface CreateDestinationRequest {
  name: string;
  type: 'ssh' | 'rclone' | 'local';
  enabled: boolean;
  
  // Type-specific fields
  host?: string;
  port?: number;
  user?: string;
  path?: string;
  remote_name?: string;
  remote_path?: string;
  
  bandwidth_limit?: number;
  concurrency?: number;
  retry_count?: number;
}

export interface ReplicateRequest {
  destination_id: string;
  snapshot_id: string;
  base_snapshot_id?: string;
}

export interface RestorePlanRequest {
  source_type: 'local' | 'ssh' | 'rclone';
  source_id: string;
  restore_type: 'full' | 'files';
  target_path: string;
}

export interface StoreSSHKeyRequest {
  key: string;
}
