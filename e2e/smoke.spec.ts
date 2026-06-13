import { test, expect } from '@playwright/test';

// Critical operator flow smoke tests
// Run against docker compose deployment

const TEST_EMAIL = 'owner@test.dev';
const TEST_PASSWORD = 'password12';

test.describe('Critical Flows', () => {

  test('login and view dashboard', async ({ page }) => {
    await page.goto('/login');
    await page.fill('input[type="email"]', TEST_EMAIL);
    await page.fill('input[type="password"]', TEST_PASSWORD);
    await page.click('button[type="submit"]');
    await page.waitForURL('/');
    await expect(page.locator('body')).toContainText('Dashboard');
  });

  test('view ops dashboard', async ({ page }) => {
    // Login first
    await page.goto('/login');
    await page.fill('input[type="email"]', TEST_EMAIL);
    await page.fill('input[type="password"]', TEST_PASSWORD);
    await page.click('button[type="submit"]');
    await page.waitForURL('/');

    // Navigate to ops
    await page.goto('/admin/ops');
    await expect(page.locator('h1')).toContainText('Operations Dashboard');
    // Should have system health section
    await expect(page.locator('body')).toContainText('System Health');
    // Should have summary section
    await expect(page.locator('body')).toContainText('Summary');
  });

  test('view integration management', async ({ page }) => {
    await page.goto('/login');
    await page.fill('input[type="email"]', TEST_EMAIL);
    await page.fill('input[type="password"]', TEST_PASSWORD);
    await page.click('button[type="submit"]');
    await page.waitForURL('/');

    await page.goto('/admin/integrations');
    await expect(page.locator('h1')).toContainText('Integration Management');
  });

  test('view admin setup checklist', async ({ page }) => {
    await page.goto('/login');
    await page.fill('input[type="email"]', TEST_EMAIL);
    await page.fill('input[type="password"]', TEST_PASSWORD);
    await page.click('button[type="submit"]');
    await page.waitForURL('/');

    await page.goto('/admin/setup');
    await expect(page.locator('h1')).toContainText('Setup');
    // Should have checklist items
    await expect(page.locator('body')).toContainText('Bootstrapped');
  });

  test('create integration key', async ({ page }) => {
    await page.goto('/login');
    await page.fill('input[type="email"]', TEST_EMAIL);
    await page.fill('input[type="password"]', TEST_PASSWORD);
    await page.click('button[type="submit"]');
    await page.waitForURL('/');

    await page.goto('/admin/integrations');
    await page.click('text=Create Key');
    await page.fill('input[placeholder*="Grafana"]', 'E2E Test Key');
    await page.fill('input[placeholder*="grafana"]', 'e2e-test');
    await page.fill('input[placeholder*="webhooks"]', 'webhooks:ingest');
    await page.click('button:has-text("Create Key")');

    // Should show the one-time credential modal
    await expect(page.locator('text=Save These Credentials')).toBeVisible({ timeout: 5000 });
  });

  test('view audit log', async ({ page }) => {
    await page.goto('/login');
    await page.fill('input[type="email"]', TEST_EMAIL);
    await page.fill('input[type="password"]', TEST_PASSWORD);
    await page.click('button[type="submit"]');
    await page.waitForURL('/');

    await page.goto('/admin/audit');
    await expect(page.locator('h1')).toContainText('Audit');
  });

  test('view agent console', async ({ page }) => {
    await page.goto('/login');
    await page.fill('input[type="email"]', TEST_EMAIL);
    await page.fill('input[type="password"]', TEST_PASSWORD);
    await page.click('button[type="submit"]');
    await page.waitForURL('/');

    await page.goto('/agents');
    // Should load without error
    await page.waitForLoadState('networkidle');
  });
});
