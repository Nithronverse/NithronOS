import { z } from 'zod'

// ============================================================================
// Error Types
// ============================================================================

export class APIError extends Error {
  constructor(
    message: string,
    public status: number,
    public code?: string,
    public details?: any,
    public retryAfterSec?: number
  ) {
    super(message)
    this.name = 'APIError'
  }
}

export class ProxyMisconfiguredError extends Error {
  constructor(
    message: string,
    public status: number,
    public contentType: string,
    public snippet: string
  ) {
    super(message)
    this.name = 'ProxyMisconfiguredError'
  }
}

// ============================================================================
// Type Schemas for Setup and Auth
// ============================================================================

export const SetupStateSchema = z.object({
  firstBoot: z.boolean(),
  otpRequired: z.boolean(),
  usersExist: z.boolean().optional(),
})

export const AuthSessionSchema = z.object({
  user: z.object({
    id: z.string(),
    username: z.string(),
    roles: z.array(z.string()),
  }).optional(),
  expiresAt: z.string().optional(),
})

export type SetupState = z.infer<typeof SetupStateSchema>
export type AuthSession = z.infer<typeof AuthSessionSchema>

// ============================================================================
// Core API Client
// ============================================================================

class ApiClient {
  private baseURL = ''
  private refreshPromise: Promise<void> | null = null
  private refreshFailures = 0

  private getCsrfToken(): string {
    const match = document.cookie.match(/(?:^|; )nos_csrf=([^;]*)/)
    return match ? decodeURIComponent(match[1]) : ''
  }

  private async refreshToken(): Promise<void> {
    // Prevent multiple simultaneous refresh attempts
    if (this.refreshPromise) return this.refreshPromise
    
    // Prevent infinite refresh loops
    if (this.refreshFailures >= 2) {
      this.refreshFailures = 0
      throw new APIError('Session expired', 401, 'auth.session.expired')
    }
    
    this.refreshPromise = fetch('/api/auth/refresh', {
      method: 'POST',
      credentials: 'include',
      headers: {
        'X-CSRF-Token': this.getCsrfToken(),
      },
    }).then(async (res) => {
      this.refreshPromise = null
      if (!res.ok) {
        this.refreshFailures++
        throw new APIError('Session expired', 401, 'auth.session.expired')
      }
      this.refreshFailures = 0
    }).catch((err) => {
      this.refreshPromise = null
      throw err
    })
    
    return this.refreshPromise
  }

  async request<T>(
    path: string,
    options: RequestInit = {},
    skipRefresh = false
  ): Promise<T> {
    const url = `${this.baseURL}${path}`
    
    // Build headers
    const headers: Record<string, string> = {
      'Accept': 'application/json',
    }
    
    // Copy existing headers
    if (options.headers) {
      if (options.headers instanceof Headers) {
        options.headers.forEach((value, key) => {
          headers[key] = value
        })
      } else if (Array.isArray(options.headers)) {
        options.headers.forEach(([key, value]) => {
          headers[key] = value
        })
      } else {
        Object.assign(headers, options.headers as Record<string, string>)
      }
    }
    
    // Add CSRF token
    const csrf = this.getCsrfToken()
    if (csrf) {
      headers['X-CSRF-Token'] = csrf
    }
    
    // Add Content-Type for body requests
    if (options.body && typeof options.body === 'string') {
      headers['Content-Type'] = 'application/json'
    }
    
    const response = await fetch(url, {
      ...options,
      headers,
      credentials: 'include',
    })

    // Check for HTML response (proxy misconfiguration)
    const contentType = response.headers.get('content-type') || ''
    if (contentType.includes('text/html')) {
      let snippet = ''
      try {
        const text = await response.text()
        snippet = text.slice(0, 200)
      } catch {}
      throw new ProxyMisconfiguredError(
        'Backend unreachable or proxy misconfigured',
        response.status,
        contentType,
        snippet
      )
    }

    // Handle 401 with token refresh (but not for auth endpoints themselves)
    if (response.status === 401 && !skipRefresh && !path.includes('/auth/')) {
      try {
        await this.refreshToken()
        // Retry once after refresh
        return this.request<T>(path, options, true)
      } catch {
        // Refresh failed, throw original error
      }
    }

    // Handle 204 No Content
    if (response.status === 204) {
      return undefined as unknown as T
    }

    // Parse error response
    if (!response.ok) {
      let message = response.statusText || `HTTP ${response.status}`
      let code: string | undefined
      let details: any
      let retryAfterSec: number | undefined
      
      try {
        if (contentType.includes('application/json')) {
          const body = await response.json()
          if (body.error) {
            message = body.error.message || body.error || message
            code = body.error.code
            details = body.error.details
            retryAfterSec = body.error.retryAfterSec
          } else if (body.message) {
            message = body.message
          }
        } else {
          const text = await response.text()
          if (text) message = text
        }
      } catch {
        // Use default message
      }
      
      throw new APIError(message, response.status, code, details, retryAfterSec)
    }

    // Parse success response
    if (contentType.includes('application/json')) {
      try {
        return await response.json()
      } catch (err) {
        throw new APIError('Invalid JSON response', 502, 'response.invalid_json')
      }
    }
    
    return undefined as unknown as T
  }

  // Helper methods
  get<T>(path: string, params?: Record<string, any>) {
    const query = params ? '?' + new URLSearchParams(params).toString() : ''
    return this.request<T>(`${path}${query}`)
  }

  post<T>(path: string, body?: any) {
    return this.request<T>(path, {
      method: 'POST',
      body: body ? JSON.stringify(body) : undefined,
    })
  }

  put<T>(path: string, body?: any) {
    return this.request<T>(path, {
      method: 'PUT',
      body: body ? JSON.stringify(body) : undefined,
    })
  }

  patch<T>(path: string, body?: any) {
    return this.request<T>(path, {
      method: 'PATCH',
      body: body ? JSON.stringify(body) : undefined,
    })
  }

  delete<T>(path: string) {
    return this.request<T>(path, { method: 'DELETE' })
  }

  // Special method for setup endpoints that need auth header
  postWithAuth<T>(path: string, token: string, body?: any) {
    return this.request<T>(path, {
      method: 'POST',
      body: body ? JSON.stringify(body) : undefined,
      headers: {
        'Authorization': `Bearer ${token}`,
      },
    })
  }
}

// ============================================================================
// Singleton Instance
// ============================================================================

export const apiClient = new ApiClient()

// ============================================================================
// Error Mapping for UI
// ============================================================================

export function getErrorMessage(error: unknown): string {
  if (error instanceof ProxyMisconfiguredError) {
    return 'Backend unreachable or proxy misconfigured'
  }
  
  if (error instanceof APIError) {
    // Map specific error codes to user-friendly messages
    switch (error.code) {
      case 'setup.complete':
        return 'Setup already completed. Please sign in.'
      case 'setup.otp.invalid':
        return 'Invalid one-time code. Please check and try again.'
      case 'setup.otp.expired':
        return 'Your code expired. Request a new one.'
      case 'setup.session.invalid':
        return 'Setup session invalid. Please restart setup.'
      case 'setup.write_failed':
        return 'Cannot write configuration. Check server permissions.'
      case 'auth.invalid_credentials':
        return 'Invalid username or password.'
      case 'auth.account_locked':
        return 'Account temporarily locked. Please try again later.'
      case 'auth.session.expired':
        return 'Session expired. Please sign in again.'
      case 'auth.csrf.invalid':
      case 'auth.csrf.missing':
        return 'Security token invalid. Please refresh and try again.'
      case 'rate.limited':
        if (error.retryAfterSec) {
          return `Too many attempts. Try again in ${error.retryAfterSec}s.`
        }
        return 'Too many attempts. Please try again later.'
      case 'input.invalid':
        return error.message || 'Invalid input. Please check your entries.'
      case 'input.weak_password':
        return 'Password too weak. Use at least 12 characters with mixed case, numbers, and symbols.'
      case 'input.username_taken':
        return 'Username already taken. Please choose another.'
      default:
        // Return the server message if available
        if (error.message && !error.message.startsWith('HTTP ')) {
          return error.message
        }
    }
    
    // Generic status messages
    switch (error.status) {
      case 400:
        return 'Invalid request. Please check your input.'
      case 401:
        return 'Authentication required. Please sign in.'
      case 403:
        return 'Access denied.'
      case 404:
        return 'Resource not found.'
      case 409:
        return 'Conflict with existing data.'
      case 410:
        return 'This action is no longer available.'
      case 429:
        return 'Too many requests. Please slow down.'
      case 500:
        return 'Server error. Please try again later.'
      case 502:
      case 503:
        return 'Service temporarily unavailable.'
      default:
        return error.message || `Request failed (${error.status})`
    }
  }
  
  if (error instanceof Error) {
    return error.message
  }
  
  return String(error || 'An unknown error occurred')
}

// ============================================================================
// API Endpoints
// ============================================================================

export const api = {
  // Setup
  setup: {
    getState: () => apiClient.get<SetupState>('/api/setup/state'),
    verifyOTP: (otp: string) => 
      apiClient.post<{ ok: boolean; token: string }>('/api/setup/verify-otp', { otp }),
    createAdmin: (token: string, data: {
      username: string
      password: string  
      enable_totp?: boolean
    }) => apiClient.postWithAuth('/api/setup/create-admin', token, data),
  },

  // Auth
  auth: {
    login: (data: {
      username: string
      password: string
      code?: string
      rememberMe?: boolean
    }) => apiClient.post<{ ok: boolean }>('/api/auth/login', data),
    
    logout: () => apiClient.post('/api/auth/logout'),
    
    refresh: () => apiClient.post('/api/auth/refresh'),
    
    session: () => apiClient.get<AuthSession>('/api/auth/me'),
    
    // TOTP
    totp: {
      enroll: () => apiClient.get<{
        otpauth_url: string
        qr_png_base64?: string
      }>('/api/auth/totp/enroll'),
      
      verify: (code: string) => apiClient.post<{
        ok: boolean
        recovery_codes?: string[]
      }>('/api/auth/totp/verify', { code }),
    },
  },
}

export default apiClient