import { test, expect, type Page, type APIRequestContext } from '@playwright/test';

// ═══════════════════════════════════════════════════════════
// ClarityIT v1.4 Track 8 — ClarityDocs E2E Smoke Tests
// Run against live deployment at http://localhost:3000
// ═══════════════════════════════════════════════════════════

const TEST_EMAIL = 'owner@test.dev';
const TEST_PASSWORD = 'password12';
const API_BASE = process.env.E2E_API_URL || 'http://localhost:8765';

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
  const loginData = await loginResp.json();
  const token = loginData.access_token;

  const meResp = await request.get(`${API_BASE}/api/auth/me`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  const meData = await meResp.json();
  return { token, teamId: meData.teams?.[0]?.id };
}

async function createTestDocument(request: APIRequestContext, token: string, teamId: string): Promise<string> {
  const resp = await request.post(`${API_BASE}/api/teams/${teamId}/artifacts/documents`, {
    headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
    data: {
      title: 'E2E Version Test',
      document_type: 'general_document',
      document_json: {
        schema_version: 1,
        title: 'E2E Version Test',
        document_type: 'general_document',
        blocks: [
          { id: 'blk_001', type: 'heading', level: 1, text: 'E2E Test Document' },
          { id: 'blk_002', type: 'paragraph', text: 'Initial content for version testing.' },
        ],
      },
    },
  });
  expect(resp.ok()).toBeTruthy();
  const data = await resp.json();
  return data.id;
}

// ═══════════════════════════════════════════════════════════
// API-only tests (no UI navigation needed)
// ═══════════════════════════════════════════════════════════

test('1. Create document creates version 1', async ({ request }) => {
  const { token, teamId } = await apiLogin(request);
  const docId = await createTestDocument(request, token, teamId);

  const resp = await request.get(`${API_BASE}/api/teams/${teamId}/artifacts/documents/${docId}/versions`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  expect(resp.ok()).toBeTruthy();
  const data = await resp.json();
  expect(data.versions.length).toBeGreaterThanOrEqual(1);
  expect(data.versions[0].version_number).toBe(1);
  expect(data.versions[0].source).toBe('user_save');
});

test('2. Version list returns DESC order', async ({ request }) => {
  const { token, teamId } = await apiLogin(request);
  const docId = await createTestDocument(request, token, teamId);

  // Patch to create v2
  await request.patch(`${API_BASE}/api/teams/${teamId}/artifacts/documents/${docId}`, {
    headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
    data: {
      document_json: {
        schema_version: 1,
        title: 'E2E Version Test',
        document_type: 'general_document',
        blocks: [{ id: 'blk_001', type: 'paragraph', text: 'v2 content' }],
      },
    },
  });

  const resp = await request.get(`${API_BASE}/api/teams/${teamId}/artifacts/documents/${docId}/versions`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  expect(resp.ok()).toBeTruthy();
  const data = await resp.json();
  expect(data.versions.length).toBe(2);
  expect(data.versions[0].version_number).toBe(2);
  expect(data.versions[1].version_number).toBe(1);
});

test('3. Restore creates new version non-destructively', async ({ request }) => {
  const { token, teamId } = await apiLogin(request);
  const docId = await createTestDocument(request, token, teamId);

  // Patch to create v2
  await request.patch(`${API_BASE}/api/teams/${teamId}/artifacts/documents/${docId}`, {
    headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
    data: {
      document_json: {
        schema_version: 1,
        title: 'E2E Version Test',
        document_type: 'general_document',
        blocks: [{ id: 'blk_001', type: 'paragraph', text: 'v2 modified' }],
      },
    },
  });

  // List versions
  const listResp = await request.get(`${API_BASE}/api/teams/${teamId}/artifacts/documents/${docId}/versions`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  const listData = await listResp.json();
  const v1Id = listData.versions.find((v: any) => v.version_number === 1).id;

  // Restore v1
  const restoreResp = await request.post(`${API_BASE}/api/teams/${teamId}/artifacts/documents/${docId}/versions/${v1Id}/restore`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  expect(restoreResp.ok()).toBeTruthy();
  const restoreData = await restoreResp.json();
  expect(restoreData.new_version_number).toBe(3);
  expect(restoreData.restored_from_version).toBe(1);

  // Verify old versions still exist (non-destructive)
  const listResp2 = await request.get(`${API_BASE}/api/teams/${teamId}/artifacts/documents/${docId}/versions`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  const listData2 = await listResp2.json();
  expect(listData2.versions.length).toBe(3);
  // v1 and v2 still present
  expect(listData2.versions.find((v: any) => v.version_number === 1)).toBeTruthy();
  expect(listData2.versions.find((v: any) => v.version_number === 2)).toBeTruthy();
  expect(listData2.versions.find((v: any) => v.version_number === 3)).toBeTruthy();
});

test('4. Version detail returns document_json', async ({ request }) => {
  const { token, teamId } = await apiLogin(request);
  const docId = await createTestDocument(request, token, teamId);

  const listResp = await request.get(`${API_BASE}/api/teams/${teamId}/artifacts/documents/${docId}/versions`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  const listData = await listResp.json();
  const v1Id = listData.versions[0].id;

  const detailResp = await request.get(`${API_BASE}/api/teams/${teamId}/artifacts/documents/${docId}/versions/${v1Id}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  expect(detailResp.ok()).toBeTruthy();
  const detail = await detailResp.json();
  expect(detail.version_number).toBe(1);
  expect(detail.document_json).toBeTruthy();
  expect(detail.document_json.blocks).toBeTruthy();
});

test('5. Cross-team version access returns 404', async ({ request }) => {
  const { token, teamId } = await apiLogin(request);
  const docId = await createTestDocument(request, token, teamId);

  const fakeTeam = '00000000-0000-0000-0000-000000000999';
  const resp = await request.get(`${API_BASE}/api/teams/${fakeTeam}/artifacts/documents/${docId}/versions`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  expect(resp.status()).toBe(404);
});

test('6. Markdown export works for native document', async ({ request }) => {
  const { token, teamId } = await apiLogin(request);
  const docId = await createTestDocument(request, token, teamId);

  const resp = await request.get(`${API_BASE}/api/teams/${teamId}/artifacts/${docId}/export/markdown`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  expect(resp.ok()).toBeTruthy();
  expect(resp.headers()['content-type']).toContain('text/markdown');
  const body = await resp.text();
  expect(body).toContain('E2E Test Document');
});

test('7. PDF export works for native document', async ({ request }) => {
  const { token, teamId } = await apiLogin(request);
  const docId = await createTestDocument(request, token, teamId);

  const resp = await request.get(`${API_BASE}/api/teams/${teamId}/artifacts/${docId}/export/pdf`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  expect(resp.ok()).toBeTruthy();
  expect(resp.headers()['content-type']).toContain('application/pdf');
  const body = await resp.body();
  expect(body[0]).toBe(0x25); // %PDF
});

test('8. DOCX export works for native document', async ({ request }) => {
  const { token, teamId } = await apiLogin(request);
  const docId = await createTestDocument(request, token, teamId);

  const resp = await request.get(`${API_BASE}/api/teams/${teamId}/artifacts/${docId}/export/docx`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  expect(resp.ok()).toBeTruthy();
  expect(resp.headers()['content-type']).toContain('wordprocessingml');
  const body = await resp.body();
  // ZIP signature: PK
  expect(body[0]).toBe(0x50); // P
  expect(body[1]).toBe(0x4B); // K
});
