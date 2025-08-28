import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import App from '../App'

// Mock backend API responses
const mockFetch = vi.fn()
global.fetch = mockFetch

describe('Auth E2E Flow', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockFetch.mockClear()
    localStorage.clear()
    sessionStorage.clear()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  const renderApp = () => {
    return render(<App />)
  }

  describe('First Boot Setup Flow', () => {
    it('should complete full setup flow from OTP to login', async () => {
      const user = userEvent.setup()
      
      // Mock setup state - first boot with OTP required
      mockFetch.mockImplementationOnce(() =>
        Promise.resolve({
          ok: true,
          json: async () => ({ firstBoot: true, otpRequired: true }),
          headers: new Headers({ 'content-type': 'application/json' }),
        })
      )
      
      renderApp()
      
      // Should redirect to setup
      await waitFor(() => {
        expect(screen.getByText(/welcome to nithronos/i)).toBeInTheDocument()
        expect(screen.getByText(/enter one-time password/i)).toBeInTheDocument()
      })
      
      // Step 1: Verify OTP
      mockFetch.mockImplementationOnce(() =>
        Promise.resolve({
          ok: true,
          json: async () => ({ success: true }),
          headers: new Headers({ 'content-type': 'application/json' }),
        })
      )
      
      await user.type(screen.getByLabelText(/one-time password/i), 'setup-otp-123')
      await user.click(screen.getByRole('button', { name: /verify/i }))
      
      // Should move to admin creation
      await waitFor(() => {
        expect(screen.getByText(/create admin account/i)).toBeInTheDocument()
      })
      
      // Step 2: Create admin account
      mockFetch.mockImplementationOnce(() =>
        Promise.resolve({
          ok: true,
          json: async () => ({ success: true, totpRequired: false }),
          headers: new Headers({ 'content-type': 'application/json' }),
        })
      )
      
      await user.type(screen.getByLabelText(/^username/i), 'admin')
      await user.type(screen.getByLabelText(/^password/i), 'AdminPassword123!')
      await user.type(screen.getByLabelText(/confirm password/i), 'AdminPassword123!')
      await user.click(screen.getByRole('button', { name: /create admin account/i }))
      
      // Should redirect to login
      await waitFor(() => {
        expect(screen.getByText(/sign in to nithronos/i)).toBeInTheDocument()
      })
      
      // Step 3: Login with created account
      mockFetch.mockImplementationOnce(() =>
        Promise.resolve({
          ok: true,
          json: async () => ({
            accessToken: 'mock-access-token',
            refreshToken: 'mock-refresh-token',
            user: { username: 'admin', roles: ['admin'] },
          }),
          headers: new Headers({ 'content-type': 'application/json' }),
        })
      )
      
      // Mock setup state - setup complete
      mockFetch.mockImplementationOnce(() =>
        Promise.resolve({
          ok: false,
          status: 410,
          headers: new Headers({ 'content-type': 'application/json' }),
        })
      )
      
      await user.type(screen.getByLabelText(/username/i), 'admin')
      await user.type(screen.getByLabelText(/password/i), 'AdminPassword123!')
      await user.click(screen.getByRole('button', { name: /sign in/i }))
      
      // Should be logged in and on dashboard
      await waitFor(() => {
        expect(screen.getByText(/dashboard/i)).toBeInTheDocument()
      }, { timeout: 5000 })
    })
  })

  describe('Session Management', () => {
    it('should refresh token on 401 and retry request', async () => {
      const user = userEvent.setup()
      
      // Mock initial login
      mockFetch
        // Setup state - not first boot
        .mockImplementationOnce(() =>
          Promise.resolve({
            ok: false,
            status: 410,
            headers: new Headers({ 'content-type': 'application/json' }),
          })
        )
        // Login request
        .mockImplementationOnce(() =>
          Promise.resolve({
            ok: true,
            json: async () => ({
              accessToken: 'initial-token',
              refreshToken: 'refresh-token',
              user: { username: 'user', roles: ['user'] },
            }),
            headers: new Headers({ 'content-type': 'application/json' }),
          })
        )
      
      renderApp()
      
      // Login
      await user.type(screen.getByLabelText(/username/i), 'user')
      await user.type(screen.getByLabelText(/password/i), 'password')
      await user.click(screen.getByRole('button', { name: /sign in/i }))
      
      // Mock API call that returns 401
      mockFetch
        // First attempt - 401
        .mockImplementationOnce(() =>
          Promise.resolve({
            ok: false,
            status: 401,
            headers: new Headers({ 'content-type': 'application/json' }),
            json: async () => ({ error: 'Token expired' }),
          })
        )
        // Refresh token
        .mockImplementationOnce(() =>
          Promise.resolve({
            ok: true,
            json: async () => ({
              accessToken: 'new-token',
              refreshToken: 'new-refresh-token',
            }),
            headers: new Headers({ 'content-type': 'application/json' }),
          })
        )
        // Retry original request
        .mockImplementationOnce(() =>
          Promise.resolve({
            ok: true,
            json: async () => ({ data: 'success' }),
            headers: new Headers({ 'content-type': 'application/json' }),
          })
        )
      
      // Should handle token refresh transparently
      await waitFor(() => {
        expect(mockFetch).toHaveBeenCalledWith(
          expect.stringContaining('/api/auth/refresh'),
          expect.any(Object)
        )
      })
    })

    it('should redirect to login when refresh fails', async () => {
      // Setup logged in state
      sessionStorage.setItem('access_token', 'expired-token')
      sessionStorage.setItem('refresh_token', 'expired-refresh')
      
      // Mock setup state check
      mockFetch.mockImplementationOnce(() =>
        Promise.resolve({
          ok: false,
          status: 410,
          headers: new Headers({ 'content-type': 'application/json' }),
        })
      )
      
      // Mock session check - 401
      mockFetch.mockImplementationOnce(() =>
        Promise.resolve({
          ok: false,
          status: 401,
          headers: new Headers({ 'content-type': 'application/json' }),
        })
      )
      
      // Mock refresh attempt - also 401
      mockFetch.mockImplementationOnce(() =>
        Promise.resolve({
          ok: false,
          status: 401,
          headers: new Headers({ 'content-type': 'application/json' }),
        })
      )
      
      renderApp()
      
      // Should redirect to login
      await waitFor(() => {
        expect(screen.getByText(/sign in to nithronos/i)).toBeInTheDocument()
        expect(screen.getByText(/session expired/i)).toBeInTheDocument()
      })
    })
  })

  describe('Logout Flow', () => {
    it('should logout and clear session', async () => {
      const user = userEvent.setup()
      
      // Setup logged in state
      sessionStorage.setItem('access_token', 'valid-token')
      sessionStorage.setItem('refresh_token', 'valid-refresh')
      
      // Mock setup state
      mockFetch.mockImplementationOnce(() =>
        Promise.resolve({
          ok: false,
          status: 410,
          headers: new Headers({ 'content-type': 'application/json' }),
        })
      )
      
      // Mock session check - return user
      mockFetch.mockImplementationOnce(() =>
        Promise.resolve({
          ok: true,
          json: async () => ({
            user: { username: 'testuser', roles: ['user'] },
          }),
          headers: new Headers({ 'content-type': 'application/json' }),
        })
      )
      
      renderApp()
      
      // Wait for dashboard to load
      await waitFor(() => {
        expect(screen.getByText(/dashboard/i)).toBeInTheDocument()
      })
      
      // Open user menu and logout
      const userMenu = screen.getByLabelText(/user menu/i)
      await user.click(userMenu)
      
      // Mock logout request
      mockFetch.mockImplementationOnce(() =>
        Promise.resolve({
          ok: true,
          headers: new Headers({ 'content-type': 'application/json' }),
        })
      )
      
      const logoutButton = screen.getByRole('button', { name: /sign out/i })
      await user.click(logoutButton)
      
      // Should clear tokens and redirect to login
      await waitFor(() => {
        expect(sessionStorage.getItem('access_token')).toBeNull()
        expect(sessionStorage.getItem('refresh_token')).toBeNull()
        expect(screen.getByText(/sign in to nithronos/i)).toBeInTheDocument()
      })
    })
  })

  describe('Backend Unreachable', () => {
    it('should show global error banner when backend is down', async () => {
      // Mock backend returning HTML (proxy error)
      mockFetch.mockImplementationOnce(() =>
        Promise.resolve({
          ok: false,
          status: 502,
          headers: new Headers({ 'content-type': 'text/html' }),
          text: async () => '<html>502 Bad Gateway</html>',
        })
      )
      
      renderApp()
      
      // Should show error banner
      await waitFor(() => {
        expect(screen.getByText(/backend unreachable/i)).toBeInTheDocument()
        expect(screen.getByRole('link', { name: /view help/i })).toHaveAttribute('href', '/help/proxy')
      })
    })
  })

  describe('Rate Limiting', () => {
    it('should handle rate limiting with retry timer', async () => {
      const user = userEvent.setup()
      
      // Mock setup state
      mockFetch.mockImplementationOnce(() =>
        Promise.resolve({
          ok: false,
          status: 410,
          headers: new Headers({ 'content-type': 'application/json' }),
        })
      )
      
      renderApp()
      
      // Attempt login that gets rate limited
      mockFetch.mockImplementationOnce(() =>
        Promise.resolve({
          ok: false,
          status: 429,
          headers: new Headers({ 
            'content-type': 'application/json',
            'retry-after': '30',
          }),
          json: async () => ({ error: 'Too many requests' }),
        })
      )
      
      await user.type(screen.getByLabelText(/username/i), 'user')
      await user.type(screen.getByLabelText(/password/i), 'password')
      await user.click(screen.getByRole('button', { name: /sign in/i }))
      
      // Should show rate limit error with retry time
      await waitFor(() => {
        expect(screen.getByText(/too many login attempts/i)).toBeInTheDocument()
        expect(screen.getByText(/try again in/i)).toBeInTheDocument()
      })
    })
  })
})
