import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { BrowserRouter } from 'react-router-dom'
import { Login } from '../pages/Login'
import { AuthProvider } from '../lib/auth'
import { api, APIError } from '../lib/api-client'

// Mock the API client
vi.mock('../lib/api-client', () => ({
  api: {
    auth: {
      login: vi.fn(),
      verifyTotp: vi.fn(),
      logout: vi.fn(),
      refresh: vi.fn(),
      getSession: vi.fn(),
      session: vi.fn().mockRejectedValue({ status: 401 }),
    },
    setup: {
      getState: vi.fn(),
    },
  },
  APIError: class APIError extends Error {
    constructor(public status: number, message: string) {
      super(message)
      this.status = status
    }
  },
  ProxyMisconfiguredError: class ProxyMisconfiguredError extends Error {
    constructor(message: string) {
      super(message)
    }
  },
  getErrorMessage: (err: any) => err.message || 'An error occurred',
}))

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
    // Default: setup is complete
    ;(api.setup.getState as any).mockRejectedValue({ status: 410, message: 'Setup complete' })
  })

  const renderLogin = () => {
    return render(
      <BrowserRouter>
        <AuthProvider>
          <Login />
        </AuthProvider>
      </BrowserRouter>
    )
  }

  it('should render login form with username and password fields', () => {
    renderLogin()
    
    expect(screen.getByLabelText(/username/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/password/i)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /sign in/i })).toBeInTheDocument()
    expect(screen.getByLabelText(/remember me/i)).toBeInTheDocument()
  })

  it('should show validation errors for empty fields', async () => {
    renderLogin()
    const user = userEvent.setup()
    
    const submitButton = screen.getByRole('button', { name: /sign in/i })
    await user.click(submitButton)
    
    await waitFor(() => {
      expect(screen.getByText(/username is required/i)).toBeInTheDocument()
      expect(screen.getByText(/password is required/i)).toBeInTheDocument()
    })
  })

  it('should handle successful login', async () => {
    vi.mocked(api.auth.login).mockResolvedValueOnce({
      ok: true,
    })
    
    renderLogin()
    const user = userEvent.setup()
    
    await user.type(screen.getByLabelText(/username/i), 'admin')
    await user.type(screen.getByLabelText(/password/i), 'password123')
    await user.click(screen.getByRole('button', { name: /sign in/i }))
    
    await waitFor(() => {
      expect(api.auth.login).toHaveBeenCalledWith({
        username: 'admin',
        password: 'password123',
        rememberMe: false,
        totpCode: undefined,
      })
      expect(mockNavigate).toHaveBeenCalledWith('/', { replace: true })
    })
  })

  it('should handle login with TOTP', async () => {
    // First attempt returns requires_totp
    const error = new APIError(401, 'code required')
    vi.mocked(api.auth.login).mockRejectedValueOnce(error)
    
    renderLogin()
    const user = userEvent.setup()
    
    await user.type(screen.getByLabelText(/username/i), 'admin')
    await user.type(screen.getByLabelText(/password/i), 'password123')
    await user.click(screen.getByRole('button', { name: /sign in/i }))
    
    // Should show TOTP field
    await waitFor(() => {
      expect(screen.getByLabelText(/two-factor authentication code/i)).toBeInTheDocument()
    })
    
    // Second attempt with TOTP
    vi.mocked(api.auth.login).mockResolvedValueOnce({
      ok: true,
    })
    
    await user.type(screen.getByLabelText(/two-factor authentication code/i), '123456')
    await user.click(screen.getByRole('button', { name: /verify/i }))
    
    await waitFor(() => {
      expect(api.auth.login).toHaveBeenCalledWith({
        username: 'admin',
        password: 'password123',
        rememberMe: false,
        totpCode: '123456',
      })
      expect(mockNavigate).toHaveBeenCalledWith('/', { replace: true })
    })
  })

  it('should handle invalid credentials error', async () => {
    const error = new APIError(401, 'Invalid username or password')
    vi.mocked(api.auth.login).mockRejectedValueOnce(error)
    
    renderLogin()
    const user = userEvent.setup()
    
    await user.type(screen.getByLabelText(/username/i), 'admin')
    await user.type(screen.getByLabelText(/password/i), 'wrong-password')
    await user.click(screen.getByRole('button', { name: /sign in/i }))
    
    await waitFor(() => {
      expect(screen.getByText(/invalid username or password/i)).toBeInTheDocument()
    })
  })

  it('should handle rate limiting error', async () => {
    const error = new APIError(429, 'Too many attempts')
    vi.mocked(api.auth.login).mockRejectedValueOnce(error)
    
    renderLogin()
    const user = userEvent.setup()
    
    await user.type(screen.getByLabelText(/username/i), 'admin')
    await user.type(screen.getByLabelText(/password/i), 'password')
    await user.click(screen.getByRole('button', { name: /sign in/i }))
    
    await waitFor(() => {
      expect(screen.getByText(/too many attempts/i)).toBeInTheDocument()
    })
  })

  it('should preserve username on failed attempts', async () => {
    const error = new APIError(401, 'Invalid username or password')
    vi.mocked(api.auth.login).mockRejectedValueOnce(error)
    
    renderLogin()
    const user = userEvent.setup()
    
    const usernameInput = screen.getByLabelText(/username/i)
    await user.type(usernameInput, 'testuser')
    await user.type(screen.getByLabelText(/password/i), 'wrong')
    await user.click(screen.getByRole('button', { name: /sign in/i }))
    
    await waitFor(() => {
      expect(usernameInput).toHaveValue('testuser')
    })
  })

  it('should handle remember me checkbox', async () => {
    vi.mocked(api.auth.login).mockResolvedValueOnce({
      ok: true,
    })
    
    renderLogin()
    const user = userEvent.setup()
    
    await user.type(screen.getByLabelText(/username/i), 'admin')
    await user.type(screen.getByLabelText(/password/i), 'password123')
    await user.click(screen.getByLabelText(/remember me/i))
    await user.click(screen.getByRole('button', { name: /sign in/i }))
    
    await waitFor(() => {
      expect(api.auth.login).toHaveBeenCalledWith({
        username: 'admin',
        password: 'password123',
        rememberMe: true,
        totpCode: undefined,
      })
    })
  })
})
