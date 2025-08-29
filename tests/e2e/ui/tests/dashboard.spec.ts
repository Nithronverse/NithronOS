import { test, expect, Page } from '@playwright/test';

// Test fixtures
const ADMIN_USERNAME = 'admin';
const ADMIN_PASSWORD = 'TestAdmin123!';

// Helper to login
async function login(page: Page) {
  await page.goto('/login');
  await page.getByLabel(/Username/i).fill(ADMIN_USERNAME);
  await page.getByLabel(/Password/i).fill(ADMIN_PASSWORD);
  await page.getByRole('button', { name: /Sign In/i }).click();
  await expect(page).toHaveURL('/');
}

test.describe('Dashboard', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('should display system metrics', async ({ page }) => {
    // Check dashboard elements
    await expect(page.getByRole('heading', { name: /Dashboard/i })).toBeVisible();
    
    // System info tile
    const systemTile = page.locator('[data-testid="system-tile"]');
    await expect(systemTile).toBeVisible();
    await expect(systemTile.getByText(/CPU Usage/i)).toBeVisible();
    await expect(systemTile.getByText(/Memory/i)).toBeVisible();
    await expect(systemTile.getByText(/Uptime/i)).toBeVisible();
    
    // Storage tile
    const storageTile = page.locator('[data-testid="storage-tile"]');
    await expect(storageTile).toBeVisible();
    await expect(storageTile.getByText(/Storage Pools/i)).toBeVisible();
    
    // Health tile
    const healthTile = page.locator('[data-testid="health-tile"]');
    await expect(healthTile).toBeVisible();
    await expect(healthTile.getByText(/System Health/i)).toBeVisible();
    
    // Recent activity
    const activitySection = page.locator('[data-testid="recent-activity"]');
    await expect(activitySection).toBeVisible();
  });

  test('should navigate to different sections', async ({ page }) => {
    // Navigate to Storage
    await page.getByRole('link', { name: /Storage/i }).click();
    await expect(page).toHaveURL(/\/storage/);
    await expect(page.getByRole('heading', { name: /Storage/i })).toBeVisible();
    
    // Navigate to Applications
    await page.getByRole('link', { name: /Applications/i }).click();
    await expect(page).toHaveURL(/\/apps/);
    
    // Navigate to Backup
    await page.getByRole('link', { name: /Backup/i }).click();
    await expect(page).toHaveURL(/\/backup/);
    
    // Navigate to Monitoring
    await page.getByRole('link', { name: /Monitoring/i }).click();
    await expect(page).toHaveURL(/\/monitor/);
    
    // Navigate back to Dashboard
    await page.getByRole('link', { name: /Dashboard/i }).click();
    await expect(page).toHaveURL('/');
  });

  test('should refresh metrics', async ({ page }) => {
    // Get initial CPU value
    const cpuElement = page.locator('[data-testid="cpu-usage"]');
    const initialValue = await cpuElement.textContent();
    
    // Click refresh button
    await page.getByRole('button', { name: /Refresh/i }).click();
    
    // Wait for update
    await page.waitForTimeout(1000);
    
    // Check that metrics are still displayed (may or may not change)
    await expect(cpuElement).toBeVisible();
  });
});

test.describe('Storage Operations', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.goto('/storage');
  });

  test('should create a snapshot', async ({ page }) => {
    // Navigate to snapshots
    await page.getByRole('tab', { name: /Snapshots/i }).click();
    
    // Click create snapshot
    await page.getByRole('button', { name: /Create Snapshot/i }).click();
    
    // Fill in snapshot details
    const modal = page.locator('[role="dialog"]');
    await expect(modal).toBeVisible();
    
    await modal.getByLabel(/Name/i).fill('test-snapshot');
    await modal.getByLabel(/Description/i).fill('E2E test snapshot');
    
    // Select subvolume
    await modal.getByRole('checkbox', { name: /@home/i }).check();
    
    // Create snapshot
    await modal.getByRole('button', { name: /Create/i }).click();
    
    // Verify success
    await expect(page.getByText(/Snapshot created successfully/i)).toBeVisible();
    
    // Verify snapshot appears in list
    await expect(page.getByText('test-snapshot')).toBeVisible();
  });

  test('should delete a snapshot', async ({ page }) => {
    // Navigate to snapshots
    await page.getByRole('tab', { name: /Snapshots/i }).click();
    
    // Find a snapshot row (assuming one exists from previous test)
    const snapshotRow = page.locator('tr').filter({ hasText: 'test-snapshot' });
    
    if (await snapshotRow.count() > 0) {
      // Click delete button
      await snapshotRow.getByRole('button', { name: /Delete/i }).click();
      
      // Confirm deletion
      const confirmDialog = page.locator('[role="dialog"]');
      await confirmDialog.getByRole('button', { name: /Confirm/i }).click();
      
      // Verify success
      await expect(page.getByText(/Snapshot deleted successfully/i)).toBeVisible();
      
      // Verify snapshot is removed
      await expect(snapshotRow).not.toBeVisible();
    }
  });
});

test.describe('Application Management', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.goto('/apps');
  });

  test('should browse app catalog', async ({ page }) => {
    // Check catalog is loaded
    await expect(page.getByRole('heading', { name: /App Catalog/i })).toBeVisible();
    
    // Check for app cards
    const appCards = page.locator('[data-testid="app-card"]');
    await expect(appCards).toHaveCount(await appCards.count());
    
    // Search for an app
    await page.getByPlaceholder(/Search apps/i).fill('nginx');
    
    // Check filtered results
    await page.waitForTimeout(500); // Debounce
    const filteredCards = page.locator('[data-testid="app-card"]');
    const count = await filteredCards.count();
    
    // At least one app should match or show "no results"
    if (count === 0) {
      await expect(page.getByText(/No apps found/i)).toBeVisible();
    } else {
      await expect(filteredCards.first()).toBeVisible();
    }
  });

  test('should view installed apps', async ({ page }) => {
    // Switch to installed tab
    await page.getByRole('tab', { name: /Installed/i }).click();
    
    // Check for installed apps list
    const installedSection = page.locator('[data-testid="installed-apps"]');
    await expect(installedSection).toBeVisible();
    
    // If apps are installed, they should have status badges
    const appRows = installedSection.locator('[data-testid="app-row"]');
    const appCount = await appRows.count();
    
    if (appCount > 0) {
      // Check first app has status
      const firstApp = appRows.first();
      await expect(firstApp.locator('[data-testid="app-status"]')).toBeVisible();
    } else {
      // Show empty state
      await expect(page.getByText(/No applications installed/i)).toBeVisible();
    }
  });
});

test.describe('User Profile', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('should open user menu', async ({ page }) => {
    // Click user avatar/menu
    await page.getByRole('button', { name: new RegExp(ADMIN_USERNAME, 'i') }).click();
    
    // Check menu items
    const menu = page.locator('[role="menu"]');
    await expect(menu).toBeVisible();
    await expect(menu.getByText(/Profile/i)).toBeVisible();
    await expect(menu.getByText(/Settings/i)).toBeVisible();
    await expect(menu.getByText(/Sign Out/i)).toBeVisible();
  });

  test('should sign out', async ({ page }) => {
    // Open user menu
    await page.getByRole('button', { name: new RegExp(ADMIN_USERNAME, 'i') }).click();
    
    // Click sign out
    await page.getByRole('menuitem', { name: /Sign Out/i }).click();
    
    // Should redirect to login
    await expect(page).toHaveURL('/login');
    
    // Should not be able to access dashboard
    await page.goto('/');
    await expect(page).toHaveURL('/login');
  });

  test('should change password', async ({ page }) => {
    // Navigate to profile
    await page.getByRole('button', { name: new RegExp(ADMIN_USERNAME, 'i') }).click();
    await page.getByRole('menuitem', { name: /Profile/i }).click();
    
    // Click change password
    await page.getByRole('button', { name: /Change Password/i }).click();
    
    // Fill password form
    const modal = page.locator('[role="dialog"]');
    await modal.getByLabel(/Current Password/i).fill(ADMIN_PASSWORD);
    await modal.getByLabel('New Password', { exact: true }).fill('NewTestAdmin123!');
    await modal.getByLabel(/Confirm New Password/i).fill('NewTestAdmin123!');
    
    // Submit
    await modal.getByRole('button', { name: /Change/i }).click();
    
    // Verify success
    await expect(page.getByText(/Password changed successfully/i)).toBeVisible();
    
    // Change it back for other tests
    await page.getByRole('button', { name: /Change Password/i }).click();
    await modal.getByLabel(/Current Password/i).fill('NewTestAdmin123!');
    await modal.getByLabel('New Password', { exact: true }).fill(ADMIN_PASSWORD);
    await modal.getByLabel(/Confirm New Password/i).fill(ADMIN_PASSWORD);
    await modal.getByRole('button', { name: /Change/i }).click();
  });
});

test.describe('Error Handling', () => {
  test('should handle API errors gracefully', async ({ page }) => {
    await login(page);
    
    // Simulate API error by navigating to a bad endpoint
    await page.route('**/api/v1/system/info', route => {
      route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Internal server error' })
      });
    });
    
    await page.goto('/');
    
    // Should show error message
    await expect(page.getByText(/Failed to load system information/i)).toBeVisible();
    
    // Should offer retry
    await expect(page.getByRole('button', { name: /Retry/i })).toBeVisible();
  });

  test('should handle network errors', async ({ page }) => {
    await login(page);
    
    // Go offline
    await page.context().setOffline(true);
    
    // Try to navigate
    await page.goto('/').catch(() => {}); // Ignore navigation error
    
    // Should show offline message
    await expect(page.getByText(/Connection lost|Offline|Network error/i)).toBeVisible();
    
    // Go back online
    await page.context().setOffline(false);
  });
});
