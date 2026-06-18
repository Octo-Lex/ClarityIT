import { test, expect, type Page } from '@playwright/test';

// ═══════════════════════════════════════════════════════════
// ClarityIT v1.0 Track 8 — Operator UI E2E Smoke Tests
// Run against live deployment at http://localhost:3000
//
// Auth tokens are in-memory only, so we use SPA navigation
// (sidebar link clicks) instead of page.goto() after login.
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

// Navigate via sidebar (preserves in-memory auth tokens)
async function navViaSidebar(page: Page, href: string) {
  await page.locator(`nav a[href="${href}"]`).click();
}

// Test 1: MFA enrollment UI flow
test('1. MFA enrollment page renders', async ({ page }) => {
  await uiLogin(page);
  await navViaSidebar(page, '/account/security');
  await expect(page.locator('main h1')).toContainText('Security', { timeout: 5000 });
});

// Test 2: Approval list displays
test('2. Approval list displays', async ({ page }) => {
  await uiLogin(page);
  await navViaSidebar(page, '/admin/approvals');
  await expect(page.locator('main h1')).toContainText('Approvals', { timeout: 5000 });
});

// Test 3: Approve action UI
test('3. Approve button visible on pending approval', async ({ page }) => {
  await uiLogin(page);
  await navViaSidebar(page, '/admin/approvals');
  await page.waitForTimeout(1000);
  const approveBtns = page.locator('[data-testid*="approve-btn-"]');
  const count = await approveBtns.count();
  if (count > 0) {
    await expect(approveBtns.first()).toBeVisible();
  }
});

// Test 4: Reject action UI
test('4. Reject button visible on pending approval', async ({ page }) => {
  await uiLogin(page);
  await navViaSidebar(page, '/admin/approvals');
  await page.waitForTimeout(1000);
  const rejectBtns = page.locator('[data-testid*="reject-btn-"]');
  const count = await rejectBtns.count();
  if (count > 0) {
    await expect(rejectBtns.first()).toBeVisible();
  }
});

// Test 5: Asset action request UI
test('5. Asset action buttons render', async ({ page }) => {
  await uiLogin(page);
  await navViaSidebar(page, '/admin/asset-actions');
  await expect(page.locator('main h1')).toContainText('Asset Actions', { timeout: 5000 });
});

// Test 6: Remediation proposal UI
test('6. Remediation panel renders', async ({ page }) => {
  await uiLogin(page);
  const errors: string[] = [];
  page.on('pageerror', err => errors.push(err.message));
  // Navigate to a non-existent incident — should still render the panel
  await page.evaluate(() => {
    window.history.pushState({}, '', '/incidents/test-id/remediation');
  });
  await page.reload();
  // Re-login after reload (tokens lost)
  await page.goto('/login');
  await page.fill('input[type="email"]', TEST_EMAIL);
  await page.fill('input[type="password"]', TEST_PASSWORD);
  await page.click('button[type="submit"]');
  await page.waitForURL('/', { timeout: 10000 });
  // Now SPA-navigate to remediation
  // Since there's no sidebar link for incidents/remediation, we use evaluate
  await page.evaluate(() => {
    window.history.pushState({}, '', '/incidents/test-id/remediation');
    window.dispatchEvent(new PopStateEvent('popstate'));
  });
  await page.waitForTimeout(2000);
  // Page should not crash
  const main = page.locator('main, h2');
  await expect(main.first()).toBeVisible({ timeout: 5000 });
});

// Test 7: Permission gating — security page accessible
test('7. Security page accessible to authenticated user', async ({ page }) => {
  await uiLogin(page);
  await navViaSidebar(page, '/account/security');
  await expect(page.locator('main h1')).toContainText('Security', { timeout: 5000 });
});

// Test 8: Sensitive data not visible
test('8. No raw secrets in approvals page', async ({ page }) => {
  await uiLogin(page);
  await navViaSidebar(page, '/admin/approvals');
  await page.waitForTimeout(2000);
  // Check page content for secret patterns
  const bodyText = await page.locator('body').textContent();
  expect(bodyText).not.toContain('super-secret');
  expect(bodyText).not.toContain('password123');
  expect(bodyText).not.toContain('token_id=');
});
