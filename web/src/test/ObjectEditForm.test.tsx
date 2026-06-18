/**
 * Direct tests for ObjectEditForm — covers the critical expected_version + 409
 * conflict contract (auto-refresh on version conflict), which is preserved
 * from the original implementation.
 *
 * Uses the vi.doMock + dynamic-import pattern (same as app.test.tsx) to grant
 * objects.update permission via a mocked auth context.
 */
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { http, HttpResponse } from 'msw';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { server } from './mockServer';
import { renderWithProviders } from './renderWithProviders';
import { setActiveTeam } from './mockServer';

const OBJECT = {
  id: 'obj-1', team_id: 'team-1', object_type: 'work_item', title: 'Rotate DB creds',
  summary: 'Quarterly rotation', status: 'open', priority: 'high', owner_id: 'u1',
  created_by: 'u1', version: 3, metadata: {}, created_at: '2026-06-17T10:00:00Z', updated_at: '2026-06-17T10:00:00Z',
};

beforeEach(() => {
  vi.resetModules();
  vi.restoreAllMocks();
  setActiveTeam();
});

describe('ObjectEditForm — expected_version + 409 contract', () => {
  it('sends expected_version and shows a conflict notice + refreshes on 409', async () => {
    let patchBody: Record<string, unknown> | undefined;
    server.use(
      http.patch('*/api/teams/:teamId/objects/obj-1', async ({ request }) => {
        patchBody = await request.json();
        return HttpResponse.json({ detail: 'version mismatch' }, { status: 409 });
      }),
    );
    // Grant objects.update so the form renders.
    vi.doMock('../auth/context', () => ({
      AuthProvider: ({ children }: { children: React.ReactNode }) => <>{children}</>,
      useAuth: () => ({ hasPermission: (p: string) => p === 'objects.update' }),
    }));
    const { default: ObjectEditForm } = await import('../features/objects/ObjectEditForm');
    const onUpdated = vi.fn();

    const user = userEvent.setup();
    renderWithProviders(
      <ObjectEditForm obj={OBJECT} onUpdated={onUpdated} onCancel={() => {}} />,
    );

    await waitFor(() => expect(screen.getByTestId('obj-save')).toBeInTheDocument());
    await user.click(screen.getByTestId('obj-save'));

    // expected_version is sent for optimistic concurrency.
    await waitFor(() => expect(patchBody).toBeDefined());
    expect(patchBody).toMatchObject({ expected_version: 3 });

    // The conflict notice surfaces and the form auto-refreshes (original UX).
    await waitFor(() => expect(screen.getByTestId('edit-conflict')).toBeInTheDocument());
    expect(screen.getByTestId('edit-conflict').textContent).toMatch(/Version conflict/i);
    await waitFor(() => expect(onUpdated).toHaveBeenCalled(), { timeout: 3000 });
  });

  it('saves successfully and calls onUpdated on a 200', async () => {
    server.use(
      http.patch('*/api/teams/:teamId/objects/obj-1', () =>
        HttpResponse.json({ message: 'updated' }),
      ),
    );
    vi.doMock('../auth/context', () => ({
      AuthProvider: ({ children }: { children: React.ReactNode }) => <>{children}</>,
      useAuth: () => ({ hasPermission: (p: string) => p === 'objects.update' }),
    }));
    const { default: ObjectEditForm } = await import('../features/objects/ObjectEditForm');
    const onUpdated = vi.fn();

    const user = userEvent.setup();
    renderWithProviders(
      <ObjectEditForm obj={OBJECT} onUpdated={onUpdated} onCancel={() => {}} />,
    );

    await user.click(screen.getByTestId('obj-save'));
    await waitFor(() => expect(onUpdated).toHaveBeenCalled());
  });

  it('shows a toast on a non-409 failure', async () => {
    server.use(
      http.patch('*/api/teams/:teamId/objects/obj-1', () =>
        HttpResponse.json({ detail: 'forbidden' }, { status: 403 }),
      ),
    );
    vi.doMock('../auth/context', () => ({
      AuthProvider: ({ children }: { children: React.ReactNode }) => <>{children}</>,
      useAuth: () => ({ hasPermission: (p: string) => p === 'objects.update' }),
    }));
    const { default: ObjectEditForm } = await import('../features/objects/ObjectEditForm');

    const user = userEvent.setup();
    renderWithProviders(
      <ObjectEditForm obj={OBJECT} onUpdated={() => {}} onCancel={() => {}} />,
    );

    await user.click(screen.getByTestId('obj-save'));
    await waitFor(() => expect(screen.getByText('Update failed')).toBeInTheDocument());
  });
});
