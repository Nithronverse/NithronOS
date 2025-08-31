import { describe, it, expect, vi, beforeEach } from 'vitest'
import { act } from 'react-dom/test-utils'

// Hoist nos-client mock BEFORE importing components under test
vi.mock('../lib/nos-client', () => {
  const auth = {
    login: vi.fn(),
    verifyTotp: vi.fn(),
    logout: vi.fn(),
    refresh: vi.fn(),
    getSession: vi.fn(),
    session: vi.fn().mockRejectedValue({ status: 401 }),
  }
  const setup = {
    getState: vi.fn(),
  }
  class APIError extends Error {
    constructor(message: string, public status: number) {
      super(message)
      this.status = status
    }
  }
  class ProxyMisconfiguredError extends Error {
    constructor(message: string) {
      super(message)
      this.name = 'ProxyMisconfiguredError'
    }
  }
  const getErrorMessage = (err: any) => err?.message || 'An error occurred'
  return {
    default: { auth, setup },
    api: { auth, setup },
    APIError,
    ProxyMisconfiguredError,
    getErrorMessage,
  }
})

// Mock AuthProvider to a no-op to avoid async effects during tests
vi.mock('../lib/auth', () => ({
  AuthProvider: ({ children }: any) => children,
  useAuth: () => ({
    session: null,
    loading: false,
    error: null,
    checkSession: vi.fn(),
    login: vi.fn(),
    logout: vi.fn(),
  }),
}))

import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { BrowserRouter } from 'react-router-dom'
import { Login } from '../pages/Login'
import http, { APIError, api } from '../lib/nos-client'

// Mock navigation
const mockNavigate = vi.fn()
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom')
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  }
})

// Mock globalNotice
vi.mock('../lib/globalNotice', () => ({
  GlobalNoticeProvider: ({ children }: any) => children,
  useGlobalNotice: () => ({
    notice: null,
    setNotice: vi.fn(),
  }),
}))

describe('Login Flow', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockNavigate.mockClear()
    // Default: setup is complete for both default and alias api
    ;(http.setup.getState as any).mockRejectedValue({ status: 410, message: 'Setup complete' })
    ;(api.setup.getState as any).mockRejectedValue({ status: 410, message: 'Setup complete' })
  })

  const renderLogin = async () => {
    await act(async () => {
      render(
        <BrowserRouter>
          {/* AuthProvider is mocked to no-op */}
          <Login />
        </BrowserRouter>
      )
    })
  }

  it('should render login form with username and password fields', async () => {
    await renderLogin()
    expect(screen.getByLabelText(/username/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/password/i)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /sign in/i })).toBeInTheDocument()
    expect(screen.getByLabelText(/remember me/i)).toBeInTheDocument()
  })

  it('should show validation errors for empty fields', async () => {
    await renderLogin()
    const user = userEvent.setup()
    const submitButton = screen.getByRole('button', { name: /sign in/i })
    await act(async () => { await user.click(submitButton) })
    await waitFor(() => {
      expect(screen.getByText(/username is required/i)).toBeInTheDocument()
      expect(screen.getByText(/password is required/i)).toBeInTheDocument()
    })
  })

  it('should handle successful login', async () => {
    vi.mocked(http.auth.login).mockResolvedValueOnce({ ok: true } as any)
    await renderLogin()
    const user = userEvent.setup()
    await act(async () => {
      await user.type(screen.getByLabelText(/username/i), 'admin')
      await user.type(screen.getByLabelText(/password/i), 'password123')
      await user.click(screen.getByRole('button', { name: /sign in/i }))
    })
    await waitFor(() => {
      expect(http.auth.login).toHaveBeenCalledWith({
        username: 'admin',
        password: 'password123',
        rememberMe: false,
        code: undefined,
      })
      expect(mockNavigate).toHaveBeenCalledWith('/', { replace: true })
    })
  })

  it('should handle invalid credentials error', async () => {
    const error = new APIError('Invalid username or password', 401)
    vi.mocked(http.auth.login).mockRejectedValueOnce(error)
    await renderLogin()
    const user = userEvent.setup()
    await act(async () => {
      await user.type(screen.getByLabelText(/username/i), 'admin')
      await user.type(screen.getByLabelText(/password/i), 'wrong-password')
      await user.click(screen.getByRole('button', { name: /sign in/i }))
    })
    await waitFor(() => {
      expect(screen.getByText(/invalid username or password/i)).toBeInTheDocument()
    })
  })

  it('should handle rate limiting error', async () => {
    const error = new APIError('Too many attempts', 429)
    ;(error as any).retryAfterSec = 10
    vi.mocked(http.auth.login).mockRejectedValueOnce(error)
    await renderLogin()
    const user = userEvent.setup()
    await act(async () => {
      await user.type(screen.getByLabelText(/username/i), 'admin')
      await user.type(screen.getByLabelText(/password/i), 'password')
      await user.click(screen.getByRole('button', { name: /sign in/i }))
    })
    await waitFor(() => {
      expect(screen.getByText(/too many attempts/i)).toBeInTheDocument()
    })
  })

  it('should preserve username on failed attempts', async () => {
    const error = new APIError('Invalid username or password', 401)
    vi.mocked(http.auth.login).mockRejectedValueOnce(error)
    await renderLogin()
    const user = userEvent.setup()
    const usernameInput = screen.getByLabelText(/username/i) as HTMLInputElement
    await act(async () => {
      await user.type(usernameInput, 'testuser')
      await user.type(screen.getByLabelText(/password/i), 'wrong')
      await user.click(screen.getByRole('button', { name: /sign in/i }))
    })
    await waitFor(() => {
      expect(usernameInput.value).toBe('testuser')
    })
  })

  it('should handle remember me checkbox', async () => {
    vi.mocked(http.auth.login).mockResolvedValueOnce({ ok: true } as any)
    await renderLogin()
    const user = userEvent.setup()
    await act(async () => {
      await user.type(screen.getByLabelText(/username/i), 'admin')
      await user.type(screen.getByLabelText(/password/i), 'password123')
      await user.click(screen.getByLabelText(/remember me/i))
      await user.click(screen.getByRole('button', { name: /sign in/i }))
    })
    await waitFor(() => {
      expect(http.auth.login).toHaveBeenCalledWith({
        username: 'admin',
        password: 'password123',
        rememberMe: true,
        code: undefined,
      })
    })
  })
})
