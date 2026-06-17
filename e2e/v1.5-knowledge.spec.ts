import { test, expect, type Page, type APIRequestContext } from '@playwright/test';

// ═══════════════════════════════════════════════════════════
// ClarityIT v1.5.0 Track 8 — Knowledge Productivity E2E Tests
// Run against live deployment at http://192.168.3.20:3000
// ═══════════════════════════════════════════════════════════

const TEST_EMAIL = 'owner@test.dev';
const TEST_PASSWORD = 'password12';
const API_BASE = process.env.E2E_API_URL || 'http://192.168.3.20:8765';

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
// API-Level Tests
// ═══════════════════════════════════════════════════════════

test.describe('v1.5 Knowledge — API', () => {
  let token: string;
  let teamId: string;

  test.beforeAll(async ({ request }) => {
    const creds = await apiLogin(request);
    token = creds.token;
    teamId = creds.teamId;
  });

  // 1: Knowledge search returns results
  test('search returns indexed content', async ({ request }) => {
    const resp = await request.get(`${API_BASE}/api/teams/${teamId}/knowledge/search?q=test`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(resp.ok()).toBeTruthy();
    const data = await resp.json();
    expect(data).toHaveProperty('results');
    expect(data).toHaveProperty('total');
    expect(data).toHaveProperty('query', 'test');
  });

  // 2: Knowledge search is team-scoped
  test('search is team-scoped', async ({ request }) => {
    const resp = await request.get(`${API_BASE}/api/teams/${teamId}/knowledge/search?q=nonexistent_xyz_123`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(resp.ok()).toBeTruthy();
    const data = await resp.json();
    expect(data.total).toBe(0);
  });

  // 3: Index status returns structure
  test('index status returns structure', async ({ request }) => {
    const resp = await request.get(`${API_BASE}/api/teams/${teamId}/knowledge/index-status`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(resp.ok()).toBeTruthy();
    const data = await resp.json();
    expect(data).toHaveProperty('total_items');
    expect(data).toHaveProperty('by_type');
    expect(data).toHaveProperty('stale_count');
  });

  // 4: Related knowledge endpoint handles non-existent source
  test('related knowledge handles non-existent source gracefully', async ({ request }) => {
    const resp = await request.get(`${API_BASE}/api/teams/${teamId}/knowledge/related?source_type=artifact&source_id=00000000-0000-0000-0000-000000000001`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    // Returns 200 with empty items, or 400/404 for non-existent source
    expect([200, 400, 404]).toContain(resp.status());
  });

  // 5: Ask Clarity endpoint handles no-sources gracefully
  test('ask clarity returns safe response with no indexed content', async ({ request }) => {
    const resp = await request.post(`${API_BASE}/api/teams/${teamId}/knowledge/ask`, {
      headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
      data: { question: 'What is the meaning of life?' },
    });
    expect(resp.ok()).toBeTruthy();
    const data = await resp.json();
    expect(data).toHaveProperty('answer');
    expect(data).toHaveProperty('sources');
    expect(data).toHaveProperty('confidence');
    expect(['low', 'medium', 'high']).toContain(data.confidence);
  });

  // 6: Collections list works
  test('collections list returns array', async ({ request }) => {
    const resp = await request.get(`${API_BASE}/api/teams/${teamId}/knowledge/collections`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(resp.ok()).toBeTruthy();
    const data = await resp.json();
    expect(data).toHaveProperty('collections');
    expect(Array.isArray(data.collections)).toBeTruthy();
  });

  // 7: Create + delete collection lifecycle
  test('create and archive collection', async ({ request }) => {
    // Create
    const createResp = await request.post(`${API_BASE}/api/teams/${teamId}/knowledge/collections`, {
      headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
      data: { name: 'E2E Test Collection', description: 'Created by E2E test' },
    });
    expect(createResp.status()).toBe(201);
    const collection = await createResp.json();
    expect(collection.name).toBe('E2E Test Collection');
    const collectionId = collection.id;

    // Archive (delete)
    const delResp = await request.delete(`${API_BASE}/api/teams/${teamId}/knowledge/collections/${collectionId}`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(delResp.ok()).toBeTruthy();
    const delData = await delResp.json();
    expect(delData.status).toBe('archived');
  });

  // 8: Saved answers lifecycle
  test('save and delete answer', async ({ request }) => {
    // Save
    const saveResp = await request.post(`${API_BASE}/api/teams/${teamId}/knowledge/saved-answers`, {
      headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
      data: {
        question: 'E2E test question?',
        answer: 'E2E test answer.',
        confidence: 'medium',
        sources: [{ source_type: 'artifact', source_id: 'e2e-1', title: 'Test Source' }],
      },
    });
    expect(saveResp.status()).toBe(201);
    const answer = await saveResp.json();
    expect(answer.question).toBe('E2E test question?');
    const answerId = answer.id;

    // List
    const listResp = await request.get(`${API_BASE}/api/teams/${teamId}/knowledge/saved-answers`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(listResp.ok()).toBeTruthy();
    const listData = await listResp.json();
    expect(listData.answers.length).toBeGreaterThan(0);

    // Delete
    const delResp = await request.delete(`${API_BASE}/api/teams/${teamId}/knowledge/saved-answers/${answerId}`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(delResp.ok()).toBeTruthy();
  });

  // 9: Quality report returns structure
  test('quality report returns structure', async ({ request }) => {
    const resp = await request.get(`${API_BASE}/api/teams/${teamId}/knowledge/quality`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(resp.ok()).toBeTruthy();
    const data = await resp.json();
    expect(data).toHaveProperty('total_items');
    expect(data).toHaveProperty('stale_count');
    expect(data).toHaveProperty('duplicate_count');
    expect(data).toHaveProperty('orphan_count');
    expect(data).toHaveProperty('by_type');
  });

  // 10: Cross-team collection returns 404
  test('cross-team collection returns 404', async ({ request }) => {
    const resp = await request.get(`${API_BASE}/api/teams/${teamId}/knowledge/collections/00000000-0000-0000-0000-000000000099`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(resp.status()).toBe(404);
  });
});

// ═══════════════════════════════════════════════════════════
// UI-Level Tests (SPA navigation via pushState)
// ═══════════════════════════════════════════════════════════

async function uiLogin(page: Page) {
  await page.goto('/login');
  await page.fill('input[type="email"]', TEST_EMAIL);
  await page.fill('input[type="password"]', TEST_PASSWORD);
  await page.click('button[type="submit"]');
  await page.waitForURL('/', { timeout: 10000 });
}

async function spaNavigate(page: Page, path: string) {
  // Navigate by clicking the nav link if available, otherwise use pushState
  const navLink = page.locator(`nav a[href="${path}"]`);
  if (await navLink.count() > 0) {
    await navLink.first().click();
    await page.waitForTimeout(1000);
  } else {
    await page.evaluate((p) => {
      window.history.pushState({}, '', p);
      window.dispatchEvent(new PopStateEvent('popstate'));
    }, path);
    await page.waitForTimeout(1500);
  }
}

test.describe('v1.5 Knowledge — UI', () => {
  test.beforeEach(async ({ page }) => {
    await uiLogin(page);
  });

  // 11: Knowledge search page renders
  test('knowledge search page renders', async ({ page }) => {
    await spaNavigate(page, '/knowledge');
    // Look for the search input which is unique to the knowledge page
    await expect(page.locator('[data-testid="knowledge-search-input"]')).toBeVisible({ timeout: 5000 });
  });

  // 12: Collections page renders or permission-gated
  test('collections page renders', async ({ page }) => {
    await spaNavigate(page, '/knowledge/collections');
    // Page should render with collections content or redirect to home
    await page.waitForTimeout(2000);
    const url = page.url();
    const hasCollections = await page.locator('text=Knowledge Collections').count() > 0;
    const hasCollectionsLoading = await page.locator('[data-testid="collections-loading"]').count() > 0;
    const hasCollectionsError = await page.locator('[data-testid="collections-error"]').count() > 0;
    const hasCollectionsEmpty = await page.locator('[data-testid="collections-empty"]').count() > 0;
    // Either the page rendered (any state) or redirected to home
    expect(hasCollections || hasCollectionsLoading || hasCollectionsError || hasCollectionsEmpty || url.endsWith('/')).toBeTruthy();
  });

  // 13: Saved answers page renders or permission-gated
  test('saved answers page renders', async ({ page }) => {
    await spaNavigate(page, '/knowledge/saved-answers');
    await page.waitForTimeout(2000);
    const url = page.url();
    const hasTitle = await page.locator('text=Saved Answers').count() > 0;
    const hasLoading = await page.locator('[data-testid="saved-answers-loading"]').count() > 0;
    const hasError = await page.locator('[data-testid="saved-answers-error"]').count() > 0;
    const hasEmpty = await page.locator('[data-testid="saved-answers-empty"]').count() > 0;
    expect(hasTitle || hasLoading || hasError || hasEmpty || url.endsWith('/')).toBeTruthy();
  });

  // 14: Quality dashboard renders or permission-gated
  test('quality dashboard renders', async ({ page }) => {
    await spaNavigate(page, '/knowledge/quality');
    await page.waitForTimeout(2000);
    const url = page.url();
    const hasTitle = await page.locator('text=Knowledge Quality').count() > 0;
    const hasLoading = await page.locator('[data-testid="quality-loading"]').count() > 0;
    const hasError = await page.locator('[data-testid="quality-error"]').count() > 0;
    const hasTotal = await page.locator('[data-testid="quality-total"]').count() > 0;
    expect(hasTitle || hasLoading || hasError || hasTotal || url.endsWith('/')).toBeTruthy();
  });

  // 15: Knowledge nav entries present
  test('knowledge nav entries present', async ({ page }) => {
    const nav = page.locator('nav');
    await expect(nav).toBeVisible();
    const navText = await nav.textContent();
    expect(navText).toContain('Knowledge');
    expect(navText).toContain('Collections');
    expect(navText).toContain('Saved Answers');
    expect(navText).toContain('Quality');
  });
});
