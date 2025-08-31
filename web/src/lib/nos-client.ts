import axios, { AxiosError, AxiosInstance } from 'axios';

export type ApiError = {
  code: string;              // e.g. "Unauthorized", "RateLimited"
  message: string;           // human-friendly
  details?: unknown;         // optional structured data
  status?: number;           // HTTP status
  requestId?: string;        // from server logs if present
};

// Backward compatibility type exports
export interface AuthSession {
  user: {
    id: string;
    username: string;
    isAdmin?: boolean;
  };
  expiresAt?: string;
}

// Backward compatibility aliases
export class APIError extends Error {
  status?: number;
  code?: string;
  retryAfterSec?: number;
  
  constructor(message: string, status?: number) {
    super(message);
    this.status = status;
  }
}

export class ProxyMisconfiguredError extends Error {
  constructor(message: string) {
    super(message);
  }
}

export function getErrorMessage(err: any): string {
  if (err?.message) return err.message;
  if (typeof err === 'string') return err;
  return 'An error occurred';
}

function readCookie(name: string): string | null {
  const m = document.cookie.match(new RegExp('(^|; )' + name.replace(/([.*+?^${}()|[\]\\])/g, '\\$1') + '=([^;]*)'));
  return m ? decodeURIComponent(m[2]) : null;
}

let refreshing: Promise<void> | null = null;

function assertV1Path(path: string) {
  if (!path.startsWith('/')) throw new Error("Path must start with '/'");
  if (path.startsWith('/api/')) {
    throw new Error("Do not include '/api' in paths; pass '/v1/...'. baseURL is already '/api'.");
  }
  if (!path.startsWith('/v1/') && !path.startsWith('/setup/')) {
    throw new Error("Path must start with '/v1/...' or '/setup/'.");
  }
}

function toApiError(err: any): ApiError {
  if (err?.isAxiosError) {
    const ae = err as AxiosError<any>;
    const status = ae.response?.status;
    const data = ae.response?.data;
    // Try to normalize server error shape; fall back sensibly
    const code = (data?.code as string) || (status === 401 ? 'Unauthorized' : status === 429 ? 'RateLimited' : 'HttpError');
    const message = (typeof data?.message === 'string' ? data.message : ae.message) || 'Request failed';
    const requestId = ae.response?.headers?.['x-request-id'];
    return { code, message, details: data?.details ?? data, status, requestId };
  }
  return { code: 'UnknownError', message: err?.message || 'Unknown error' };
}

const client: AxiosInstance = axios.create({
  baseURL: '/api',
  withCredentials: true,
  timeout: 15000,
  headers: { 'Accept': 'application/json' }
});

// Request interceptor: add CSRF header if cookie exists
client.interceptors.request.use((config) => {
  const csrf = readCookie('nos_csrf') || readCookie('csrf_token');
  if (csrf) {
    (config.headers ||= {})['X-CSRF-Token'] = csrf;
  }
  // Guard path correctness for relative URLs
  const url = (config.url || '');
  if (url.startsWith('/')) assertV1Path(url);
  return config;
});

// Response interceptor: on 401, try single refresh then replay once
client.interceptors.response.use(
  (res) => res,
  async (error) => {
    const err = toApiError(error);
    const original = error?.config;
    if (err.status === 401 && !original?._retry) {
      if (!refreshing) {
        refreshing = (async () => {
          try { await client.post('/v1/auth/refresh', {}); }
          finally { refreshing = null; }
        })();
      }
      await refreshing;
      original._retry = true;
      return client(original);
    }
    // For backward compatibility, throw APIError instances
    const apiErr = new APIError(err.message, err.status);
    apiErr.code = err.code;
    if ((err as any).retryAfterSec) apiErr.retryAfterSec = (err as any).retryAfterSec;
    return Promise.reject(apiErr);
  }
);

// Minimal typed wrapper
const httpCore = {
  async get<T>(path: string, params?: Record<string, any>): Promise<T> {
    assertV1Path(path);
    const res = await client.get<T>(path, { params });
    return res.data;
  },
  async post<T>(path: string, body?: any): Promise<T> {
    assertV1Path(path);
    const res = await client.post<T>(path, body ?? {});
    return res.data;
  },
  async put<T>(path: string, body?: any): Promise<T> {
    assertV1Path(path);
    const res = await client.put<T>(path, body ?? {});
    return res.data;
  },
  async patch<T>(path: string, body?: any): Promise<T> {
    assertV1Path(path);
    const res = await client.patch<T>(path, body ?? {});
    return res.data;
  },
  async del<T>(path: string): Promise<T> {
    assertV1Path(path);
    const res = await client.delete<T>(path);
    return res.data;
  },
};

// Extended API client with nested structure for backward compatibility
export const http = {
  ...httpCore,
  
  // Setup endpoints
  setup: {
    getState: () => httpCore.get<any>('/setup/state'),
    verifyOTP: (otp: string) => httpCore.post('/setup/verify-otp', { otp }),
    createAdmin: (token: string, data: any) => httpCore.post('/setup/create-admin', { token, ...data }),
    complete: (token: string) => httpCore.post('/setup/complete', { token }),
  },
  
  // Auth endpoints
  auth: {
    login: (data: any) => httpCore.post('/v1/auth/login', data),
    logout: () => httpCore.post('/v1/auth/logout'),
    refresh: () => httpCore.post('/v1/auth/refresh', {}),
    me: () => httpCore.get('/v1/auth/me'),
    verifyTotp: (code: string) => httpCore.post('/v1/auth/verify-totp', { code }),
    getSession: () => httpCore.get('/v1/auth/session'),
    session: () => httpCore.get('/v1/auth/session'),
    totp: {
      enroll: () => httpCore.post('/v1/auth/totp/enroll'),
      verify: (code: string) => httpCore.post('/v1/auth/totp/verify', { code }),
    },
  },
  
  // System endpoints
  system: {
    getTimezone: (token?: string) => httpCore.get('/v1/system/timezone'),
    getNTP: (token?: string) => httpCore.get('/v1/system/ntp'),
    setHostname: (data: any, token?: string) => httpCore.post('/v1/system/hostname', data),
    setTimezone: (data: any, token?: string) => httpCore.post('/v1/system/timezone', data),
    setNTP: (data: any, token?: string) => httpCore.post('/v1/system/ntp', data),
  },
  
  // Network endpoints
  network: {
    getInterfaces: (token?: string) => httpCore.get('/v1/network/interfaces'),
    configureInterface: (iface: string, config: any) => httpCore.post(`/v1/network/interfaces/${iface}`, config),
  },
  
  // Telemetry endpoints
  telemetry: {
    setConsent: (data: any) => httpCore.post('/v1/telemetry/consent', data),
  },
  
  // Pool endpoints
  pools: {
    get: (id: string) => httpCore.get(`/v1/pools/${id}`),
    list: () => httpCore.get('/v1/pools'),
    getMountOptions: (id: string) => httpCore.get(`/v1/pools/${id}/mount-options`),
    updateMountOptions: (id: string, options: any) => httpCore.post(`/v1/pools/${id}/mount-options`, options),
    planDevice: (id: string, body: any) => httpCore.post(`/v1/pools/${id}/plan-device`, body),
    applyDevice: (id: string, body: any) => httpCore.post(`/v1/pools/${id}/apply-device`, body),
    roots: () => httpCore.get('/v1/pools/roots'),
  },
  
  // Updates endpoints
  updates: {
    check: () => httpCore.get('/v1/updates/check'),
    apply: (data: any) => httpCore.post('/v1/updates/apply', data),
    rollback: (data: any) => httpCore.post('/v1/updates/rollback', data),
  },
  
  // Shares endpoints
  shares: {
    create: (data: any) => httpCore.post('/v1/shares', data),
  },
  
  // SMB endpoints
  smb: {
    usersList: () => httpCore.get('/v1/smb/users'),
    userCreate: (data: any) => httpCore.post('/v1/smb/users', data),
  },
};

// Backward compatibility - export api as an alias to http
export const api = http;

// Re-export the base http methods as default
export default http;
