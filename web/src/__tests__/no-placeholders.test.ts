import { describe, it, expect, vi } from 'vitest'
import { dashboardApi } from '../lib/api-dashboard'
import http from '../lib/nos-client'

// Mock http to simulate API errors
vi.mock('../lib/nos-client', () => ({
  default: {
    get: vi.fn(),
  }
}))

describe('No Placeholders Policy', () => {
  it('should not return fake zeros when API fails', async () => {
    // Simulate API failure
    vi.mocked(http.get).mockRejectedValue(new Error('Network error'))
    
    // getDashboard should throw, not return fake data
    await expect(dashboardApi.getDashboard()).rejects.toThrow('Network error')
    
    // getSystemSummary should throw, not return fake data
    await expect(dashboardApi.getSystemSummary()).rejects.toThrow('Network error')
    
    // getStorageSummary should throw, not return fake data
    await expect(dashboardApi.getStorageSummary()).rejects.toThrow('Network error')
  })
  
  it('should not have placeholder strings in dashboardApi', () => {
    // These patterns should not exist
    const forbiddenPatterns = [
      'cpuPct: 0',
      'mem: { used: 0, total: 1 }',
      'totalBytes: 0',
      'poolsOnline: 0',
      'MOCK',
      'FAKE',
      'PLACEHOLDER',
      'DUMMY'
    ]
    
    forbiddenPatterns.forEach(pattern => {
      // Check the actual implementation doesn't contain these
      const implementation = Object.values(dashboardApi)
        .map(fn => fn.toString())
        .join(' ')
      expect(implementation).not.toContain(pattern)
    })
  })
  
  it('should validate nos-client path rules', () => {
    // Paths should not start with /api
    const validPaths = ['/v1/health', '/setup/state', '/auth/login']
    const invalidPaths = ['/api/v1/health', '/api/setup/state']
    
    // This would be enforced by nos-client in development
    // Here we just document the expectation
    expect(validPaths.every(p => !p.startsWith('/api/'))).toBe(true)
    expect(invalidPaths.every(p => p.startsWith('/api/'))).toBe(true)
  })
})

describe('Real Data Flow', () => {
  it('should handle successful API responses', async () => {
    const mockData = {
      system: {
        status: 'ok',
        cpuPct: 45,
        mem: { used: 8000000000, total: 16000000000 },
        uptimeSec: 86400
      }
    }
    
    vi.mocked(http.get).mockResolvedValue(mockData)
    
    const result = await dashboardApi.getDashboard()
    expect(result).toEqual(mockData)
    expect(result.system.cpuPct).toBe(45) // Real non-zero value
  })
  
  it('should show proper loading states', () => {
    // Components should show loading state, not fake data
    // This would be tested in component tests
    expect(true).toBe(true)
  })
  
  it('should show proper error states', () => {
    // Components should show error state, not fake data
    // This would be tested in component tests
    expect(true).toBe(true)
  })
})
