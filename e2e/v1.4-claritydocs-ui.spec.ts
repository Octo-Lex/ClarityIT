import { test, expect, type Page, type APIRequestContext } from '@playwright/test';

// ═══════════════════════════════════════════════════════════
// ClarityIT v1.4 Track 8 — ClarityDocs Version History
// UI Drawer Playwright Spec (Track 7 carry-forward closure)
//
// Drives the REAL SPA UI: opens the editor, clicks History,
// selects a prior version, previews it, restores it, and
// verifies content + new version + absence of dangerous controls.
//
// Auth tokens are in-memory only — after uiLogin we use
// history.pushState + popstate for SPA navigation to preserve
// the session token without a full page reload.
// ═══════════════════════════════════════════════════════════

const TEST_EMAIL = 'owner@test.dev';
const TEST_PASSWORD = 'password12';
const API_BASE = process.env.E2E_API_URL || 'http://192.168.3.20:8765';

// ─── Helpers ───

async function uiLogin(page: Page) {
  await page.goto('/login');
  await page.fill('input[type="email"]', TEST_EMAIL);
  await page.fill('input[type="password"]', TEST_PASSWORD);
  await page.click('button[type="submit"]');
  await page.waitForURL('/', { timeout: 10000 });
  await expect(page.locator('nav')).toBeVisible({ timeout: 5000 });
}

async function apiLogin(request: APIRequestContext): Promise<{ token: string; teamId: string }> {
  const loginResp = await request.post(`${API_BASE}/api/auth/login`, {
    data: { email: TEST_EMAIL, password: TEST_PASSWORD },
  });
  expect(loginResp.ok()).toBeTruthy();
  const { access_token: token } = await loginResp.json();
  const meResp = await request.get(`${API_BASE}/api/auth/me`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  const meData = await meResp.json();
  return { token, teamId: meData.teams[0].id };
}

/** Navigate within the SPA without losing the in-memory auth token. */
async function spaNavigate(page: Page, path: string) {
  await page.evaluate((p) => {
    window.history.pushState({}, '', p);
    window.dispatchEvent(new PopStateEvent('popstate'));
  }, path);
}

/** Create a document with original content (v1), then patch it (v2). */
async function seedDocumentWithVersions(
  request: APIRequestContext,
  token: string,
  teamId: string,
): Promise<{ artifactId: string; updatedAt: string }> {
  // Create — v1
  const createResp = await request.post(`${API_BASE}/api/teams/${teamId}/artifacts/documents`, {
    headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
    data: {
      title: 'Restore Flow Test',
      document_type: 'general_document',
      document_json: {
        schema_version: 1,
        title: 'Restore Flow Test',
        document_type: 'general_document',
        blocks: [
          { id: 'h1', type: 'heading', level: 1, text: 'Original Heading' },
          { id: 'p1', type: 'paragraph', text: 'This is the original paragraph content from version one.' },
        ],
      },
    },
  });
  expect(createResp.ok()).toBeTruthy();
  const doc = await createResp.json();

  // Patch — v2 (different content)
  const patchResp = await request.patch(`${API_BASE}/api/teams/${teamId}/artifacts/documents/${doc.id}`, {
    headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
    data: {
      title: 'Restore Flow Test',
      document_json: {
        schema_version: 1,
        title: 'Restore Flow Test',
        document_type: 'general_document',
        blocks: [
          { id: 'h1', type: 'heading', level: 1, text: 'Updated Heading' },
          { id: 'p1', type: 'paragraph', text: 'This is the updated content from version two.' },
        ],
      },
    },
  });
  expect(patchResp.ok()).toBeTruthy();
  const patched = await patchResp.json();
  return { artifactId: doc.id, updatedAt: patched.updated_at };
}

// ═══════════════════════════════════════════════════════════
// Test: Version History Drawer — full UI restore flow
// ═══════════════════════════════════════════════════════════

test('Version History drawer: open, preview, restore, verify content and absence of controls', async ({ page, request }) => {
  // ── Setup: login via API, seed document with 2 versions ──
  const { token, teamId } = await apiLogin(request);
  const { artifactId } = await seedDocumentWithVersions(request, token, teamId);

  // ── Step 1: Login via UI and navigate to the document editor ──
  await uiLogin(page);
  await spaNavigate(page, `/artifacts/documents/${artifactId}`);

  // Wait for the editor page to render
  await expect(page.locator('[data-testid="doc-editor-page"]')).toBeVisible({ timeout: 10000 });

  // ── Step 2: Verify editor loaded with v2 content ──
  await expect(page.locator('[data-testid="doc-title-input"]')).toHaveValue('Restore Flow Test');
  // v2 heading should be visible
  const editorArea = page.locator('[data-testid="doc-editor-page"]');
  await expect(editorArea).toContainText('Updated Heading');

  // ── Step 3: Click Version History button ──
  await page.locator('[data-testid="version-history-btn"]').click();

  // ── Step 4: Assert drawer is visible ──
  await expect(page.locator('[data-testid="version-drawer"]')).toBeVisible({ timeout: 5000 });

  // ── Step 5: Assert version list renders ──
  await expect(page.locator('[data-testid="version-list"]')).toBeVisible();
  // v2 should be at the top (DESC order)
  await expect(page.locator('[data-testid="version-item-2"]')).toBeVisible();
  // v1 should be below
  await expect(page.locator('[data-testid="version-item-1"]')).toBeVisible();

  // Verify DESC order: v2 appears before v1 in the DOM
  const v2Box = await page.locator('[data-testid="version-item-2"]').boundingBox();
  const v1Box = await page.locator('[data-testid="version-item-1"]').boundingBox();
  expect(v2Box!.y).toBeLessThan(v1Box!.y);

  // ── Step 6: Select prior version (v1) ──
  await page.locator('[data-testid="version-item-1"]').click();

  // ── Step 7: Assert preview renders with v1 content ──
  await expect(page.locator('[data-testid="version-preview"]')).toBeVisible({ timeout: 5000 });
  const previewText = await page.locator('[data-testid="version-preview"]').textContent();
  expect(previewText).toContain('Original Heading');
  expect(previewText).toContain('original paragraph content');

  // ── Step 8: Click Restore ──
  await expect(page.locator('[data-testid="restore-button"]')).toBeVisible();
  await page.locator('[data-testid="restore-button"]').click();

  // ── Step 9: Confirm restore ──
  await expect(page.locator('[data-testid="restore-confirm"]')).toBeVisible({ timeout: 3000 });
  await expect(page.locator('[data-testid="restore-confirm-button"]')).toBeVisible();
  await page.locator('[data-testid="restore-confirm-button"]').click();

  // ── Step 10: Wait for restore to complete and version list to reload ──
  // After restore, loadVersions() runs and v3 (source: restore) should appear at top
  await expect(page.locator('[data-testid="version-item-3"]')).toBeVisible({ timeout: 10000 });

  // Verify v3 is at the top (DESC order)
  const v3Box = await page.locator('[data-testid="version-item-3"]').boundingBox();
  const v2BoxAfter = await page.locator('[data-testid="version-item-2"]').boundingBox();
  expect(v3Box!.y).toBeLessThan(v2BoxAfter!.y);

  // Verify v3 has "Restored" badge
  await expect(page.locator('[data-testid="version-badge-3"]')).toContainText('Restored');

  // ── Step 11: Close drawer and verify restored content in editor ──
  await page.locator('[data-testid="close-drawer"]').click();
  await expect(page.locator('[data-testid="version-drawer"]')).not.toBeVisible({ timeout: 3000 });

  // The editor should now show v1 content (restored)
  const editorAfter = page.locator('[data-testid="doc-editor-page"]');
  await expect(editorAfter).toContainText('Original Heading', { timeout: 5000 });
  await expect(editorAfter).toContainText('original paragraph content');

  // ── Step 12: Re-open drawer to verify version count and absence of controls ──
  await page.locator('[data-testid="version-history-btn"]').click();
  await expect(page.locator('[data-testid="version-drawer"]')).toBeVisible({ timeout: 5000 });

  // Assert 3 versions exist (v3 at top)
  await expect(page.locator('[data-testid="version-item-3"]')).toBeVisible();
  await expect(page.locator('[data-testid="version-item-2"]')).toBeVisible();
  await expect(page.locator('[data-testid="version-item-1"]')).toBeVisible();

  // ── Step 13: Assert NO dangerous controls exist in the drawer ──
  // No delete, share, approval, or execute controls
  const drawer = page.locator('[data-testid="version-drawer"]');
  await expect(drawer.locator('[data-testid="version-delete"]')).toHaveCount(0);
  await expect(drawer.locator('[data-testid="version-share"]')).toHaveCount(0);
  await expect(drawer.locator('[data-testid="version-approve"]')).toHaveCount(0);
  await expect(drawer.locator('[data-testid="version-execute"]')).toHaveCount(0);
  await expect(drawer.locator('text=Delete')).toHaveCount(0);
  await expect(drawer.locator('text=Share')).toHaveCount(0);
  await expect(drawer.locator('text=Approve')).toHaveCount(0);
  await expect(drawer.locator('text=Execute')).toHaveCount(0);
});
