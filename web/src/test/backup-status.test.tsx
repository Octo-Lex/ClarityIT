import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';

// Mock the API client with backup status
vi.mock('../api/client', () => ({
  api: {
    opsSummary: vi.fn().mockResolvedValue({ outbox_pending: 0, dead_letters: 0, agent_runs_pending: 0, agent_runs_running: 0, webhook_rejections_24h: 0, agent_blocks_24h: 0, security_events_24h: 0, integration_keys_active: 2, integration_keys_rotation_required: 0, total_users: 3, total_teams: 1 }),
    deepHealth: vi.fn().mockResolvedValue(null),
    opsWorkers: vi.fn().mockResolvedValue([]),
    opsDeadLetters: vi.fn().mockResolvedValue([]),
    opsWebhookRejections: vi.fn().mockResolvedValue([]),
    opsAgentBlocks: vi.fn().mockResolvedValue([]),
    backupStatus: vi.fn(),
  },
}));

import AdminOps from '../features/admin/AdminOps';
import { api } from '../api/client';

describe('AdminOps Backup Status', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders backup status card with recent backups', async () => {
    vi.mocked(api.backupStatus).mockResolvedValue({
      postgres: {
        last_backup_at: new Date().toISOString(),
        size_bytes: 364544,
        path: 'postgresql_20260614_100000.sql.gz',
        age_status: 'green',
      },
      minio: {
        last_backup_at: new Date().toISOString(),
        size_bytes: 8192,
        path: 'minio_20260614_100200.tar.gz',
        age_status: 'green',
      },
      restore_drill: {
        last_verified_at: new Date().toISOString(),
        status: 'passed',
        source: 'restore-drill-report',
      },
    });

    render(
      <MemoryRouter>
        <AdminOps />
      </MemoryRouter>
    );

    await waitFor(() => {
      expect(screen.getByText('Backup Status')).toBeInTheDocument();
    });

    expect(screen.getByText('PostgreSQL')).toBeInTheDocument();
    expect(screen.getByText('MinIO')).toBeInTheDocument();
    expect(screen.getByText('Restore Drill')).toBeInTheDocument();
    expect(screen.getAllByText('green').length).toBeGreaterThan(0);
    expect(screen.getByText('passed')).toBeInTheDocument();
  });

  it('renders missing backup state', async () => {
    vi.mocked(api.backupStatus).mockResolvedValue({
      postgres: { last_backup_at: '', size_bytes: 0, path: '', age_status: 'missing' },
      minio: { last_backup_at: '', size_bytes: 0, path: '', age_status: 'missing' },
      restore_drill: { last_verified_at: '', status: 'unknown', source: 'restore-drill-report' },
    });

    render(
      <MemoryRouter>
        <AdminOps />
      </MemoryRouter>
    );

    await waitFor(() => {
      expect(screen.getByText('Backup Status')).toBeInTheDocument();
    });

    expect(screen.getAllByText('No backup found').length).toBe(2);
    expect(screen.getAllByText('missing').length).toBeGreaterThan(0);
  });

  it('does not render backup or restore action buttons', async () => {
    vi.mocked(api.backupStatus).mockResolvedValue({
      postgres: { last_backup_at: new Date().toISOString(), size_bytes: 1024, path: 'pg.sql.gz', age_status: 'green' },
      minio: { last_backup_at: new Date().toISOString(), size_bytes: 1024, path: 'minio.tar.gz', age_status: 'green' },
      restore_drill: { last_verified_at: new Date().toISOString(), status: 'passed', source: 'restore-drill-report' },
    });

    render(
      <MemoryRouter>
        <AdminOps />
      </MemoryRouter>
    );

    await waitFor(() => {
      expect(screen.getByText('Backup Status')).toBeInTheDocument();
    });

    // Verify no backup/restore trigger buttons exist
    expect(screen.queryByText(/create backup/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/run backup/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/trigger restore/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/run restore drill/i)).not.toBeInTheDocument();
  });
});
