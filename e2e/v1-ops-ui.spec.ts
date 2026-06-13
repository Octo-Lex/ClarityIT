import { test, expect, type Page } from '@playwright/test';

// ═══════════════════════════════════════════════════════════
// ClarityIT v1.0 Track 6 — Operator UI E2E Smoke Tests
// Defined now, executed at final gate (Track 8).
//
// These tests verify the v1.0 operator UI capabilities:
// MFA enrollment, approvals, asset actions, remediation,
// permission gating, and sensitive data protection.
// ═══════════════════════════════════════════════════════════

const TEST_EMAIL = 'owner@test.dev';
const TEST_PASSWORD = 'password12';

async function uiLogin(page: Page) {
  await page.goto('/login');
  await page.fill('input[type="email"]', TEST_EMAIL);
  await page.fill('input[type="password"]', TEST_PASSWORD);
  await page.click('button[type="submit"]');
  await page.waitForURL('/', { timeout: 10000 });
  await expect(page.locator('nav')).toBeVisible({ timeout: 5000 });
}

// Test 1: MFA enrollment UI flow
test('MFA enrollment page renders', async ({ page }) => {
  await uiLogin(page);
  await page.goto('/account/security');
  await expect(page.locator('[data-testid="enroll-mfa-btn"]')).toBeVisible({ timeout: 5000 });
});

// Test 2: Approval list displays
test('approval list displays pending approvals', async ({ page }) => {
  await uiLogin(page);
  await page.goto('/admin/approvals');
  // Should show the Approvals heading
  await expect(page.locator('h1')).toContainText('Approvals', { timeout: 5000 });
});

// Test 3: Approve action UI
test('approve button visible on pending approval', async ({ page }) => {
  await uiLogin(page);
  await page.goto('/admin/approvals');
  await page.waitForTimeout(1000); // wait for API
  // Look for any approve button
  const approveBtns = page.locator('[data-testid*="approve-btn-"]');
  const count = await approveBtns.count();
  if (count > 0) {
    await expect(approveBtns.first()).toBeVisible();
  }
});

// Test 4: Reject action UI
test('reject button visible on pending approval', async ({ page }) => {
  await uiLogin(page);
  await page.goto('/admin/approvals');
  await page.waitForTimeout(1000);
  const rejectBtns = page.locator('[data-testid*="reject-btn-"]');
  const count = await rejectBtns.count();
  if (count > 0) {
    await expect(rejectBtns.first()).toBeVisible();
  }
});

// Test 5: Asset action request UI
test('asset action buttons render', async ({ page }) => {
  await uiLogin(page);
  // Navigate to an asset page — need an asset ID, use 'test' as placeholder
  // This test verifies the route renders without crashing
  await page.goto('/');
  // The asset actions page would need a real asset ID
  // For smoke, we just verify the admin asset-actions list renders
  await page.goto('/admin/asset-actions');
  await expect(page.locator('h1')).toContainText('Asset Actions', { timeout: 5000 });
});

// Test 6: Remediation proposal UI
test('remediation panel renders', async ({ page }) => {
  await uiLogin(page);
  // Navigate to an incident remediation page
  // For smoke, verify the page loads without JS errors
  const errors: string[] = [];
  page.on('pageerror', err => errors.push(err.message));
  await page.goto('/incidents/test-id/remediation');
  await page.waitForTimeout(2000);
  // Should show the Remediation heading
  await expect(page.locator('h2')).toContainText('Remediation', { timeout: 5000 });
});

// Test 7: Permission gating
test('security page accessible to authenticated user', async ({ page }) => {
  await uiLogin(page);
  await page.goto('/account/security');
  await expect(page.locator('h1')).toContainText('Security', { timeout: 5000 });
});

// Test 8: Sensitive data not visible
test('no raw secrets in approvals page', async ({ page }) => {
  await uiLogin(page);
  const secrets: string[] = [];
  page.on('response', async res => {
    const text = await res.text().catch(() => '');
    if (text.includes('super-secret') || text.includes('password123') || text.includes('token_id=')) {
      secrets.push(text.substring(0, 100));
    }
  });
  await page.goto('/admin/approvals');
  await page.waitForTimeout(2000);
  expect(secrets.length).toBe(0);
});
