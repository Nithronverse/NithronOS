// API Response Types

export interface SystemInfo {
  version: string;
  kernel: string;
  hostname: string;
  arch: string;
  memoryTotal: number;
  uptime: number;
  cpuUsage?: number;
  memoryUsage?: number;
}

export interface DiskInfo {
  disks: Array<{
    path: string;
    fstype: string;
    mountpoint: string;
    size: number;
    used?: number;
    available?: number;
  }>;
}

export interface PoolInfo {
  id: string;
  mount: string;
  uuid: string;
  label: string;
  size: number;
  used: number;
  status: string;
  devices?: string[];
}

export interface ShareInfo {
  name: string;
  path: string;
  protocol: string;
  enabled: boolean;
  description?: string;
  guestOk?: boolean;
  readOnly?: boolean;
  users?: string[];
  groups?: string[];
}

export interface SnapshotInfo {
  id: string;
  name: string;
  path: string;
  created: string;
  size?: number;
}

export interface UpdateInfo {
  version: string;
  channel: string;
  available: boolean;
  current_version?: string;
  new_version?: string;
  release_notes?: string;
}

export interface JobInfo {
  id: string;
  type: string;
  status: string;
  progress?: number;
  error?: string;
  started_at?: string;
  completed_at?: string;
}

export interface RemoteStats {
  totalBackupSize: number;
  successRate: number;
}

export interface DestinationInfo {
  id: string;
  name: string;
  type: string;
  status: string;
  endpoint?: string;
}

export interface DeviceInfo {
  path: string;
  name: string;
  type: string;
  size: number;
  model?: string;
  serial?: string;
  smart?: SmartData;
}

export interface SmartData {
  status: string;
  temperature?: number;
  powerOnHours?: number;
  attributes?: Array<{
    id: number;
    name: string;
    value: number;
    worst: number;
    threshold: number;
    raw: string;
  }>;
}

export interface TxLogResponse {
  lines: string[];
  nextCursor: number;
}

export interface AuthSession {
  user: {
    id: string;
    username: string;
    isAdmin?: boolean;
    roles?: string[];
  };
  token?: string;
  expiresAt?: string;
}

export interface SetupState {
  firstBoot: boolean;
  token?: string;
  step?: string;
  completed?: boolean;
}

export interface OTPEnrollResponse {
  qr_png_base64?: string;
  otpauth_url?: string;
  secret?: string;
}

export interface OTPVerifyResponse {
  recovery_codes?: string[];
  token?: string;
}

export interface NetworkInterface {
  name: string;
  addresses: string[];
  mtu: number;
  state: string;
}

export interface NetworkInterfacesResponse {
  interfaces: NetworkInterface[];
}

export interface TimezoneResponse {
  timezone: string;
  available?: string[];
}

export interface NTPResponse {
  enabled: boolean;
  servers?: string[];
}

export interface PruneResponse {
  ok: boolean;
  pruned?: number;
  errors?: string[];
}
