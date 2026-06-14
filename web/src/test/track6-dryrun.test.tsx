import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { AuthProvider } from '../auth/context';

// Mock the API client
vi.mock('../api/client', () => ({
  api: {
    me: vi.fn().mockResolvedValue({ id: 'u1', email: 'owner@test.dev', name: 'Owner', teams: [{ id: 't1', name: 'Team', slug: 'team', role: 'owner' }] }),
    permissions: vi.fn().mockResolvedValue({ role: 'owner', team_id: 't1', permissions: ['assets.actions.create', 'assets.actions.execute'] }),
    createAssetAction: vi.fn(),
    dryRunAssetAction: vi.fn(),
    listAssetActions: vi.fn().mockResolvedValue([]),
  },
  setAccessToken: vi.fn(),
  getStoredTeamId: vi.fn().mockReturnValue('t1'),
  setStoredTeamId: vi.fn(),
  ApiError: class extends Error { constructor(public status: number, msg: string) { super(msg); } },
}));

import AssetActions from '../features/assets/AssetActions';
import { api } from '../api/client';

function renderWithProviders(ui: React.ReactElement) {
  return render(
    <MemoryRouter>
      <AuthProvider>{ui}</AuthProvider>
    </MemoryRouter>
  );
}

describe('AssetActions — Dry-Run Preview', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  // Test 1: Preview button renders on asset action page
  it('renders Preview buttons on the asset action page', async () => {
    renderWithProviders(<AssetActions assetId="asset-1" hostname="test-vm" />);

    await waitFor(() => {
      expect(screen.getByTestId('btn-preview-start')).toBeInTheDocument();
    });
    expect(screen.getByTestId('btn-preview-snapshot')).toBeInTheDocument();
    expect(screen.getByTestId('btn-preview-shutdown')).toBeInTheDocument();
    expect(screen.getByTestId('btn-preview-stop')).toBeInTheDocument();
  });

  // Test 2: Preview calls API with dry_run=true
  it('calls dryRunAssetAction when Preview button is clicked', async () => {
    const mockPreview = {
      dry_run: true,
      action_type: 'proxmox.start',
      target: { asset_id: 'asset-1', name: 'test-vm', provider: 'proxmox', node: 'pve1', vmid: 100, vm_type: 'qemu' },
      risk_level: 'medium',
      requires_approval: true,
      requires_mfa: false,
      min_approvers: 1,
      mutation_window_required: true,
      mutation_window_active: false,
      feature_flag_enabled: false,
      would_create_approval: true,
      would_create_asset_action: false,
      would_call_proxmox: false,
      validation: { asset_valid: true, target_valid: true, snapshot_name_valid: true, policy_valid: true },
    };
    vi.mocked(api.dryRunAssetAction).mockResolvedValue(mockPreview);

    renderWithProviders(<AssetActions assetId="asset-1" hostname="test-vm" />);

    await waitFor(() => {
      expect(screen.getByTestId('btn-preview-start')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByTestId('btn-preview-start'));

    await waitFor(() => {
      expect(api.dryRunAssetAction).toHaveBeenCalledWith('asset-1', 'start', undefined);
    });
  });

  // Test 3: Preview shows risk/MFA/approval/min-approver data
  it('displays risk, MFA, approval, and min-approver data', async () => {
    const mockPreview = {
      dry_run: true,
      action_type: 'proxmox.stop',
      target: { asset_id: 'asset-1', name: 'test-vm', provider: 'proxmox', node: 'pve1', vmid: 100, vm_type: 'qemu' },
      risk_level: 'critical',
      requires_approval: true,
      requires_mfa: true,
      min_approvers: 2,
      mutation_window_required: true,
      mutation_window_active: false,
      feature_flag_enabled: false,
      would_create_approval: true,
      would_create_asset_action: false,
      would_call_proxmox: false,
      validation: { asset_valid: true, target_valid: true, snapshot_name_valid: true, policy_valid: true },
    };
    vi.mocked(api.dryRunAssetAction).mockResolvedValue(mockPreview);

    renderWithProviders(<AssetActions assetId="asset-1" hostname="test-vm" />);

    await waitFor(() => {
      expect(screen.getByTestId('btn-preview-stop')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByTestId('btn-preview-stop'));

    await waitFor(() => {
      expect(screen.getByTestId('preview-result')).toBeInTheDocument();
    });

    expect(screen.getByTestId('preview-risk').textContent).toBe('critical');
    expect(screen.getByTestId('preview-mfa').textContent).toBe('Yes');
    expect(screen.getByTestId('preview-approval').textContent).toBe('Yes');
    expect(screen.getByTestId('preview-min-approvers').textContent).toBe('2');
  });

  // Test 4: Preview shows mutation window and feature flag status
  it('displays mutation window and feature flag status', async () => {
    const mockPreview = {
      dry_run: true,
      action_type: 'proxmox.start',
      target: { asset_id: 'asset-1', name: 'test-vm', provider: 'proxmox', node: 'pve1', vmid: 100, vm_type: 'qemu' },
      risk_level: 'medium',
      requires_approval: true,
      requires_mfa: false,
      min_approvers: 1,
      mutation_window_required: true,
      mutation_window_active: false,
      feature_flag_enabled: false,
      would_create_approval: true,
      would_create_asset_action: false,
      would_call_proxmox: false,
      validation: { asset_valid: true, target_valid: true, snapshot_name_valid: true, policy_valid: true },
    };
    vi.mocked(api.dryRunAssetAction).mockResolvedValue(mockPreview);

    renderWithProviders(<AssetActions assetId="asset-1" hostname="test-vm" />);

    await waitFor(() => {
      expect(screen.getByTestId('btn-preview-start')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByTestId('btn-preview-start'));

    await waitFor(() => {
      expect(screen.getByTestId('preview-mutation-window')).toBeInTheDocument();
    });

    expect(screen.getByTestId('preview-mutation-window').textContent).toContain('Inactive');
    expect(screen.getByTestId('preview-feature-flag').textContent).toContain('Disabled');
    expect(screen.getByTestId('preview-blocked-status').textContent).toContain('Blocked');
  });

  // Test 5: Preview states no action was requested/executed
  it('shows no-action warning message', async () => {
    const mockPreview = {
      dry_run: true,
      action_type: 'proxmox.start',
      target: { asset_id: 'asset-1', name: 'test-vm', provider: 'proxmox', node: 'pve1', vmid: 100, vm_type: 'qemu' },
      risk_level: 'medium',
      requires_approval: true,
      requires_mfa: false,
      min_approvers: 1,
      mutation_window_required: true,
      mutation_window_active: true,
      feature_flag_enabled: true,
      would_create_approval: true,
      would_create_asset_action: false,
      would_call_proxmox: false,
      validation: { asset_valid: true, target_valid: true, snapshot_name_valid: true, policy_valid: true },
    };
    vi.mocked(api.dryRunAssetAction).mockResolvedValue(mockPreview);

    renderWithProviders(<AssetActions assetId="asset-1" hostname="test-vm" />);

    await waitFor(() => {
      expect(screen.getByTestId('btn-preview-start')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByTestId('btn-preview-start'));

    await waitFor(() => {
      expect(screen.getByTestId('preview-warning')).toBeInTheDocument();
    });

    expect(screen.getByTestId('preview-warning').textContent).toContain('Preview only');
    expect(screen.getByTestId('preview-warning').textContent).toContain('no action has been requested or executed');
  });

  // Test 6: Invalid preview error renders safely
  it('renders preview error safely', async () => {
    vi.mocked(api.dryRunAssetAction).mockRejectedValue(new Error('connection failed'));

    renderWithProviders(<AssetActions assetId="bad-asset" hostname="test-vm" />);

    await waitFor(() => {
      expect(screen.getByTestId('btn-preview-start')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByTestId('btn-preview-start'));

    await waitFor(() => {
      expect(screen.getByTestId('preview-error')).toBeInTheDocument();
    });

    const errEl = screen.getByTestId('preview-error');
    // Should show the error message (generic fallback)
    expect(errEl.textContent).toContain('Failed to preview');
    // Should NOT contain any sensitive material
    expect(errEl.textContent).not.toContain('token');
    expect(errEl.textContent).not.toContain('secret');
  });

  // Test 7: No sensitive metadata rendered
  it('does not render sensitive target metadata', async () => {
    const mockPreview = {
      dry_run: true,
      action_type: 'proxmox.start',
      target: { asset_id: 'asset-1', name: 'test-vm', provider: 'proxmox', node: 'pve1', vmid: 100, vm_type: 'qemu' },
      risk_level: 'medium',
      requires_approval: true,
      requires_mfa: false,
      min_approvers: 1,
      mutation_window_required: true,
      mutation_window_active: true,
      feature_flag_enabled: true,
      would_create_approval: true,
      would_create_asset_action: false,
      would_call_proxmox: false,
      validation: { asset_valid: true, target_valid: true, snapshot_name_valid: true, policy_valid: true },
    };
    vi.mocked(api.dryRunAssetAction).mockResolvedValue(mockPreview);

    const { container } = renderWithProviders(<AssetActions assetId="asset-1" hostname="test-vm" />);

    await waitFor(() => {
      expect(screen.getByTestId('btn-preview-start')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByTestId('btn-preview-start'));

    await waitFor(() => {
      expect(screen.getByTestId('preview-result')).toBeInTheDocument();
    });

    const fullHTML = container.innerHTML;
    // Should not render any internal/secret material
    expect(fullHTML).not.toContain('external_id');
    expect(fullHTML).not.toContain('public_key');
    expect(fullHTML).not.toContain('token');
    expect(fullHTML).not.toContain('secret');
    expect(fullHTML).not.toContain('challenge');
  });
});
