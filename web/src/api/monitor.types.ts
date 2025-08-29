// Monitoring API Types

export interface SystemMetrics {
  timestamp: string;
  cpu: CPUMetrics;
  memory: MemoryMetrics;
  load: LoadMetrics;
  uptime_seconds: number;
}

export interface CPUMetrics {
  usage_percent: number;
  core_usage?: number[];
  temperature?: number;
}

export interface MemoryMetrics {
  total: number;
  used: number;
  free: number;
  available: number;
  used_percent: number;
  swap_total: number;
  swap_used: number;
  swap_percent: number;
}

export interface LoadMetrics {
  load1: number;
  load5: number;
  load15: number;
}

export interface DiskMetrics {
  device: string;
  mount_point?: string;
  total: number;
  used: number;
  free: number;
  used_percent: number;
  read_bytes: number;
  write_bytes: number;
  read_ops: number;
  write_ops: number;
  temperature?: number;
  smart_status?: string;
}

export interface NetworkMetrics {
  interface: string;
  rx_bytes: number;
  tx_bytes: number;
  rx_packets: number;
  tx_packets: number;
  rx_errors: number;
  tx_errors: number;
  rx_dropped: number;
  tx_dropped: number;
  link_state: string;
  speed_mbps?: number;
}

export interface ServiceMetrics {
  name: string;
  state: string;
  sub_state: string;
  active: boolean;
  running: boolean;
  since?: string;
  main_pid?: number;
  memory_bytes?: number;
  cpu_percent?: number;
  restart_count?: number;
}

export interface BtrfsMetrics {
  device: string;
  mount_point: string;
  scrub_state?: string;
  scrub_progress?: number;
  last_scrub?: string;
  errors_write: number;
  errors_read: number;
  errors_flush: number;
  errors_corrupt: number;
  errors_generation: number;
}

export interface MonitorOverview {
  timestamp: string;
  system: SystemMetrics;
  disks: DiskMetrics[];
  network: NetworkMetrics[];
  services: ServiceMetrics[];
  btrfs?: BtrfsMetrics[];
  alerts_active: number;
}

export interface TimeSeries {
  metric: string;
  labels?: Record<string, string>;
  data_points: DataPoint[];
}

export interface DataPoint {
  timestamp: string;
  value: number;
  labels?: Record<string, string>;
}

export interface TimeSeriesQuery {
  metric: string;
  start_time: string;
  end_time: string;
  step?: number;
  filters?: Record<string, string>;
  aggregate?: 'avg' | 'min' | 'max' | 'sum';
}

// Alert types

export type AlertSeverity = 'info' | 'warning' | 'critical';

export interface AlertRule {
  id: string;
  name: string;
  description?: string;
  enabled: boolean;
  
  // Condition
  metric: string;
  operator: '>' | '<' | '==' | '!=' | '>=' | '<=';
  threshold: number;
  duration: number; // seconds
  filters?: Record<string, string>;
  
  // Alert properties
  severity: AlertSeverity;
  cooldown: number; // seconds
  hysteresis?: number;
  
  // Notification
  channels: string[];
  template?: string;
  
  // Metadata
  created_at: string;
  updated_at: string;
  last_fired?: string;
  last_cleared?: string;
  
  // State
  current_state: RuleState;
}

export interface RuleState {
  firing: boolean;
  since?: string;
  value: number;
  last_checked: string;
}

export interface NotificationChannel {
  id: string;
  name: string;
  type: 'email' | 'webhook' | 'ntfy';
  enabled: boolean;
  config: Record<string, any>;
  
  // Rate limiting
  rate_limit?: number;
  quiet_hours?: QuietHours;
  
  // Metadata
  created_at: string;
  updated_at: string;
  last_used?: string;
  last_error?: string;
}

export interface QuietHours {
  enabled: boolean;
  start_time: string;
  end_time: string;
  weekends: boolean;
}

export interface AlertEvent {
  id: string;
  rule_id: string;
  rule_name: string;
  severity: AlertSeverity;
  state: 'firing' | 'cleared';
  
  // Alert details
  metric: string;
  value: number;
  threshold: number;
  message: string;
  
  // Timestamps
  fired_at: string;
  cleared_at?: string;
  
  // Notification status
  notified: boolean;
  channels?: string[];
  notify_error?: string;
}

// Channel configs

export interface EmailConfig {
  smtp_host: string;
  smtp_port: number;
  smtp_user?: string;
  smtp_password?: string;
  use_tls: boolean;
  use_starttls: boolean;
  from: string;
  to: string[];
  subject?: string;
}

export interface WebhookConfig {
  url: string;
  method?: string;
  headers?: Record<string, string>;
  template?: string;
  secret?: string;
}

export interface NtfyConfig {
  server_url: string;
  topic: string;
  priority?: number;
  tags?: string[];
  username?: string;
  password?: string;
  token?: string;
}
