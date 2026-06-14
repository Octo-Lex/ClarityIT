import { test, expect, type Page, type APIRequestContext } from '@playwright/test';

// ═══════════════════════════════════════════════════════════
// ClarityIT v0.9.0 Operator Readiness — E2E Smoke Tests
// Run against live deployment at http://192.168.3.20:3000
//
// Auth tokens are in-memory only (not localStorage), so after
// login we use SPA navigation (sidebar link clicks) instead
// of page.goto() to preserve the session.
// ═══════════════════════════════════════════════════════════

const TEST_EMAIL = 'owner@test.dev';
const TEST_PASSWORD = 'password12';
const API_BASE = process.env.E2E_API_URL || 'http://192.168.3.20:8765';

// Login via UI and wait for dashboard to load.
async function uiLogin(page: Page) {
  await page.goto('/login');
  await page.fill('input[type="email"]', TEST_EMAIL);
  await page.fill('input[type="password"]', TEST_PASSWORD);
  await page.click('button[type="submit"]');
  await page.waitForURL('/', { timeout: 10000 });
  await expect(page.locator('nav')).toBeVisible({ timeout: 5000 });
}

// Navigate via sidebar link click (SPA navigation — preserves auth)
async function navViaSidebar(page: Page, href: string) {
  await page.locator(`nav a[href="${href}"]`).click();
}

// Login via API and return token + teamId
async function apiLogin(request: APIRequestContext): Promise<{ token: string; teamId: string }> {
  const loginResp = await request.post(`${API_BASE}/api/auth/login`, {
    data: { email: TEST_EMAIL, password: TEST_PASSWORD },
  });
  expect(loginResp.ok()).toBeTruthy();
  const loginData = await loginResp.json();
  const token = loginData.access_token;

  const meResp = await request.get(`${API_BASE}/api/auth/me`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  const meData = await meResp.json();
  return { token, teamId: meData.teams?.[0]?.id };
}

// ═══════════════════════════════════════════════════════════
// Test 1: Login and view dashboard
// ═══════════════════════════════════════════════════════════
test('1. Login and view dashboard', async ({ page }) => {
  await uiLogin(page);
  await expect(page.locator('main h1')).toContainText('Dashboard');
});

// ═══════════════════════════════════════════════════════════
// Test 2: Create work item
// ═══════════════════════════════════════════════════════════
test('2. Create work item', async ({ page }) => {
  await uiLogin(page);

  // Navigate: Queue → "+ New"
  await navViaSidebar(page, '/queue');
  await page.waitForSelector('table');
  await page.click('a:has-text("+ New"), button:has-text("+ New")');
  await page.waitForURL('/work-items/new');
  await expect(page.locator('main h1')).toContainText('New Work Item');

  // Fill and submit
  const title = `E2E Test Item ${Date.now()}`;
  await page.fill('input[placeholder="Title *"]', title);
  await page.fill('textarea[placeholder="Summary"]', 'Created by Playwright E2E test');
  await page.click('button[type="submit"]');

  // Should redirect to object detail page
  await page.waitForURL('/objects/*', { timeout: 10000 });
  await expect(page.locator('main h1')).toContainText(title);
});

// ═══════════════════════════════════════════════════════════
// Test 3: View board
// ═══════════════════════════════════════════════════════════
test('3. View board', async ({ page }) => {
  await uiLogin(page);
  await navViaSidebar(page, '/board');
  await page.waitForLoadState('networkidle');

  await expect(page.locator('main h1')).toContainText('Board');
  await expect(page.locator('text=Open')).toBeVisible();
  await expect(page.locator('text=Blocked')).toBeVisible();
});

// ═══════════════════════════════════════════════════════════
// Test 4: Open object detail
// ═══════════════════════════════════════════════════════════
test('4. Open object detail', async ({ page, request }) => {
  // Ensure at least one work item exists via API
  const { token, teamId } = await apiLogin(request);
  await request.post(`${API_BASE}/api/teams/${teamId}/work-items`, {
    headers: {
      Authorization: `Bearer ${token}`,
      'Content-Type': 'application/json',
      'Idempotency-Key': `e2e-setup-${Date.now()}`,
    },
    data: {
      title: `E2E Detail Test ${Date.now()}`,
      summary: 'Test object for detail view',
      work_item_type: 'task',
      status: 'open',
      priority: 'medium',
    },
  });

  await uiLogin(page);
  await navViaSidebar(page, '/queue');
  // Wait for actual data rows (not "No items found")
  await page.waitForSelector('table tbody tr[style*="cursor"], table tbody tr.clickable, table tbody tr', { timeout: 5000 });
  await page.locator('table tbody tr').first().click();
  await page.waitForURL('/objects/*', { timeout: 10000 });

  await expect(page.locator('text=Details')).toBeVisible();
  await expect(page.locator('text=Comments')).toBeVisible();
});

// ═══════════════════════════════════════════════════════════
// Test 5: Add comment
// ═══════════════════════════════════════════════════════════
test('5. Add comment to object', async ({ page, request }) => {
  // Ensure at least one work item exists
  const { token, teamId } = await apiLogin(request);
  const createResp = await request.post(`${API_BASE}/api/teams/${teamId}/work-items`, {
    headers: {
      Authorization: `Bearer ${token}`,
      'Content-Type': 'application/json',
      'Idempotency-Key': `e2e-comment-${Date.now()}`,
    },
    data: {
      title: `E2E Comment Test ${Date.now()}`,
      summary: 'Test object for commenting',
      work_item_type: 'task',
      status: 'open',
      priority: 'low',
    },
  });
  const obj = await createResp.json();

  await uiLogin(page);
  await navViaSidebar(page, '/queue');
  await page.waitForSelector('table tbody tr', { timeout: 5000 });
  await page.locator('table tbody tr').first().click();
  await page.waitForURL('/objects/*', { timeout: 10000 });

  // Add comment
  const commentText = `E2E comment ${Date.now()}`;
  await page.fill('input[placeholder="Add a comment..."]', commentText);
  await page.click('button:has-text("Post")');

  // Comment should appear
  await expect(page.locator(`text=${commentText}`)).toBeVisible({ timeout: 5000 });
});

// ═══════════════════════════════════════════════════════════
// Test 6: Create integration key
// ═══════════════════════════════════════════════════════════
test('6. Create integration key', async ({ page }) => {
  await uiLogin(page);
  await navViaSidebar(page, '/admin/integrations');
  await expect(page.locator('main h1')).toContainText('Integration Management');

  // Open create form
  await page.click('button:has-text("Create Key")');
  await page.waitForSelector('input[placeholder*="Grafana"]', { timeout: 3000 });

  // Fill form
  await page.fill('input[placeholder*="Grafana"]', `E2E Key ${Date.now()}`);
  await page.fill('input[placeholder*="grafana"]', 'grafana');
  await page.fill('input[placeholder*="webhooks"]', 'webhooks:ingest');

  // Submit (use the last button matching — the form submit, not toggle)
  await page.locator('button:has-text("Create Key")').last().click();

  // Should show the one-time credential modal
  await expect(page.locator('text=Save These Credentials')).toBeVisible({ timeout: 5000 });
  await expect(page.locator('code').first()).toContainText('clarity_');
});

// ═══════════════════════════════════════════════════════════
// Test 7: Send signed webhook test (API-level)
// ═══════════════════════════════════════════════════════════
test('7. Send signed webhook test', async ({ request }) => {
  const { token, teamId } = await apiLogin(request);

  // Create integration key
  const keyResp = await request.post(`${API_BASE}/api/teams/${teamId}/integration-keys`, {
    headers: {
      Authorization: `Bearer ${token}`,
      'Content-Type': 'application/json',
      'Idempotency-Key': `e2e-webhook-${Date.now()}`,
    },
    data: {
      name: 'E2E Webhook Test',
      allowed_sources: ['grafana'],
      allowed_scopes: ['webhooks:ingest'],
    },
  });
  expect(keyResp.ok()).toBeTruthy();
  const keyData = await keyResp.json();
  const rawKey = keyData.key;
  expect(rawKey).toBeTruthy();

  // Create HMAC-SHA256 signature using rawKey as the signing secret
  // (server uses rawKey as HMAC key — see verifySignature in handler.go)
  const timestamp = Math.floor(Date.now() / 1000).toString();
  const payload = {
    name: 'E2E Alert Test',
    severity: 'warning',
    source_id: 'e2e-test',
    message: 'Test alert from Playwright E2E',
  };
  const bodyStr = JSON.stringify(payload);
  const signingString = `${timestamp}.${bodyStr}`;

  const crypto = await import('crypto');
  const signature = crypto.createHmac('sha256', rawKey).update(signingString).digest('hex');

  // Send signed webhook — use raw string body so it matches signature exactly
  const webhookResp = await request.post(`${API_BASE}/api/webhooks/grafana`, {
    headers: {
      'X-ClarityIT-Integration-Key': rawKey,
      'X-ClarityIT-Signature': `v1=${signature}`,
      'X-ClarityIT-Timestamp': timestamp,
      'Content-Type': 'application/json',
    },
    data: bodyStr,
  });
  expect(webhookResp.status()).toBe(201);

  // Unsigned webhook should be rejected (allow_unsigned_dev=false)
  const unsignedResp = await request.post(`${API_BASE}/api/webhooks/grafana`, {
    headers: {
      'X-ClarityIT-Integration-Key': rawKey,
      'Content-Type': 'application/json',
    },
    data: bodyStr,
  });
  expect(unsignedResp.status()).toBe(401);
});

// ═══════════════════════════════════════════════════════════
// Test 8: View queue
// ═══════════════════════════════════════════════════════════
test('8. View queue page', async ({ page }) => {
  await uiLogin(page);
  await navViaSidebar(page, '/queue');

  await expect(page.locator('main h1')).toContainText('Queue');
  await expect(page.locator('table')).toBeVisible();
});

// ═══════════════════════════════════════════════════════════
// Test 9: Open agent console
// ═══════════════════════════════════════════════════════════
test('9. Open agent console', async ({ page }) => {
  await uiLogin(page);
  await navViaSidebar(page, '/agents');
  await page.waitForLoadState('networkidle');
  await page.waitForTimeout(1000);

  // Page should load without error
  const mainContent = page.locator('main');
  await expect(mainContent).toBeVisible({ timeout: 5000 });
  const text = await mainContent.textContent();
  expect(text!.length).toBeGreaterThan(0);
});

// ═══════════════════════════════════════════════════════════
// Test 10: View ops dashboard
// ═══════════════════════════════════════════════════════════
test('10. View ops dashboard', async ({ page }) => {
  await uiLogin(page);
  await navViaSidebar(page, '/admin/ops');
  await page.waitForLoadState('networkidle');

  await expect(page.locator('main h1')).toContainText('Operations Dashboard');
  await expect(page.locator('h2:has-text("System Health")')).toBeVisible();
  await expect(page.locator('.capitalize:has-text("postgres")')).toBeVisible();
  await expect(page.locator('h2:has-text("Summary")')).toBeVisible();
  await expect(page.locator('text=Outbox Pending')).toBeVisible();
  await expect(page.locator('h2:has-text("Worker Status")')).toBeVisible();
  await expect(page.locator('h2:has-text("Dead Letters")')).toBeVisible();
});

// ═══════════════════════════════════════════════════════════
// Test 11: View admin setup checklist
// ═══════════════════════════════════════════════════════════
test('11. View admin setup checklist', async ({ page }) => {
  await uiLogin(page);
  await navViaSidebar(page, '/admin/setup');

  await expect(page.locator('main h1')).toContainText('Setup');
  await expect(page.locator('text=Bootstrapped')).toBeVisible();
});
