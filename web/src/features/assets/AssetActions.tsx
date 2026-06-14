import { useState } from 'react';
import { api, ApiError } from '../../api/client';
import { usePermissions } from '../../hooks/usePermissions';

interface AssetActionsProps {
  assetId: string;
  hostname?: string;
}

interface DryRunPreview {
  dry_run: boolean;
  action_type: string;
  target: {
    asset_id: string;
    name: string;
    provider: string;
    node: string;
    vmid: number;
    vm_type: string;
  };
  risk_level: string;
  requires_approval: boolean;
  requires_mfa: boolean;
  min_approvers: number;
  mutation_window_required: boolean;
  mutation_window_active: boolean;
  feature_flag_enabled: boolean;
  would_create_approval: boolean;
  would_create_asset_action: boolean;
  would_call_proxmox: boolean;
  validation: {
    asset_valid: boolean;
    target_valid: boolean;
    snapshot_name_valid: boolean;
    policy_valid: boolean;
  };
}

export default function AssetActions({ assetId, hostname }: AssetActionsProps) {
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [snapshotName, setSnapshotName] = useState('');
  const [busy, setBusy] = useState<string | null>(null);
  const [preview, setPreview] = useState<DryRunPreview | null>(null);
  const [previewError, setPreviewError] = useState('');
  const { hasPermission } = usePermissions();

  const canCreate = hasPermission('assets.actions.create') || hasPermission('assets.actions.request');
  const canExecute = hasPermission('assets.actions.execute');

  const handleAction = async (action: string) => {
    setError(''); setSuccess(''); setBusy(action);
    try {
      if (action === 'snapshot' && !snapshotName) {
        setError('Snapshot name is required');
        setBusy(null);
        return;
      }
      // Validate snapshot name client-side
      if (action === 'snapshot' && snapshotName && !/^[a-zA-Z0-9_-]{1,40}$/.test(snapshotName)) {
        setError('Snapshot name: only alphanumeric, hyphens, underscores (max 40 chars)');
        setBusy(null);
        return;
      }
      const result = await api.createAssetAction(assetId, action, snapshotName || undefined);
      setSuccess(`Action "${action}" created — approval pending. ID: ${result.id?.slice(0, 8)}...`);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : `Failed to create ${action} action`);
    }
    setBusy(null);
  };

  const handlePreview = async (action: string) => {
    setPreview(null); setPreviewError(''); setBusy('preview-' + action);
    try {
      if (action === 'snapshot' && snapshotName && !/^[a-zA-Z0-9_-]{1,40}$/.test(snapshotName)) {
        setPreviewError('Snapshot name: only alphanumeric, hyphens, underscores (max 40 chars)');
        setBusy(null);
        return;
      }
      const result = await api.dryRunAssetAction(assetId, action, snapshotName || undefined);
      setPreview(result);
    } catch (e) {
      setPreviewError(e instanceof ApiError ? e.message : `Failed to preview ${action} action`);
    }
    setBusy(null);
  };

  const riskColor = (risk: string) => {
    switch (risk) {
      case 'medium': return 'text-yellow-400';
      case 'high': return 'text-orange-400';
      case 'critical': return 'text-red-400';
      default: return 'text-[var(--text-muted)]';
    }
  };

  if (!canCreate && !hasPermission('assets.actions.read')) return null;

  return (
    <div className="mt-4 p-4 bg-[var(--card)] border border-[var(--border)] rounded-lg" data-testid="asset-actions-container">
      <h3 className="text-sm font-semibold mb-3">Proxmox Actions {hostname ? `— ${hostname}` : ''}</h3>

      {error && <div className="mb-3 p-2 bg-red-900/30 border border-red-700 rounded text-xs text-red-300">{error}</div>}
      {success && <div className="mb-3 p-2 bg-green-900/20 border border-green-700 rounded text-xs text-green-300">{success}</div>}

      <div className="grid grid-cols-2 gap-2" data-testid="asset-action-buttons">
        {/* Start */}
        <div>
          <button
            onClick={() => handleAction('start')}
            disabled={!canCreate || busy !== null}
            data-testid="btn-start"
            className="w-full px-3 py-2 bg-green-900/30 border border-green-700 rounded text-sm text-green-300 hover:bg-green-900/50 disabled:opacity-50"
          >
            ▶ Start
            <span className="block text-xs text-[var(--text-muted)]">Medium risk • Approval required</span>
          </button>
          <button
            onClick={() => handlePreview('start')}
            disabled={busy !== null}
            data-testid="btn-preview-start"
            className="w-full mt-1 px-2 py-1 text-xs text-[var(--text-muted)] hover:text-[var(--text)] underline"
          >
            Preview
          </button>
        </div>

        {/* Snapshot */}
        <div>
          <button
            onClick={() => handleAction('snapshot')}
            disabled={!canCreate || busy !== null || (!!snapshotName && !/^[a-zA-Z0-9_-]{1,40}$/.test(snapshotName))}
            data-testid="btn-snapshot"
            className="w-full px-3 py-2 bg-blue-900/30 border border-blue-700 rounded text-sm text-blue-300 hover:bg-blue-900/50 disabled:opacity-50"
          >
            📸 Snapshot
            <span className="block text-xs text-[var(--text-muted)]">Medium risk • Approval required</span>
          </button>
          <button
            onClick={() => handlePreview('snapshot')}
            disabled={busy !== null}
            data-testid="btn-preview-snapshot"
            className="w-full mt-1 px-2 py-1 text-xs text-[var(--text-muted)] hover:text-[var(--text)] underline"
          >
            Preview
          </button>
          <input
            type="text"
            placeholder="snapshot name (optional)"
            value={snapshotName}
            onChange={e => setSnapshotName(e.target.value)}
            maxLength={40}
            className="w-full mt-1 px-2 py-1 bg-[var(--bg)] border border-[var(--border)] rounded text-xs"
          />
        </div>

        {/* Shutdown */}
        <div>
          <button
            onClick={() => handleAction('shutdown')}
            disabled={!canCreate || busy !== null}
            data-testid="btn-shutdown"
            className="w-full px-3 py-2 bg-orange-900/30 border border-orange-700 rounded text-sm text-orange-300 hover:bg-orange-900/50 disabled:opacity-50"
          >
            ⏹ Shutdown
            <span className="block text-xs text-[var(--text-muted)]">High risk • MFA required</span>
          </button>
          <button
            onClick={() => handlePreview('shutdown')}
            disabled={busy !== null}
            data-testid="btn-preview-shutdown"
            className="w-full mt-1 px-2 py-1 text-xs text-[var(--text-muted)] hover:text-[var(--text)] underline"
          >
            Preview
          </button>
        </div>

        {/* Stop — elevated */}
        <div>
          <button
            onClick={() => handleAction('stop')}
            disabled={!canCreate || busy !== null}
            data-testid="btn-stop"
            className="w-full px-3 py-2 bg-red-900/40 border border-red-700 rounded text-sm text-red-300 hover:bg-red-900/60 disabled:opacity-50"
          >
            ⛔ Stop (Force)
            <span className="block text-xs text-red-400 font-medium" data-testid="stop-warning">
              Critical • 2 approvers + MFA required
            </span>
          </button>
          <button
            onClick={() => handlePreview('stop')}
            disabled={busy !== null}
            data-testid="btn-preview-stop"
            className="w-full mt-1 px-2 py-1 text-xs text-[var(--text-muted)] hover:text-[var(--text)] underline"
          >
            Preview
          </button>
        </div>
      </div>

      {/* Preview Result */}
      {previewError && (
        <div className="mt-3 p-2 bg-red-900/30 border border-red-700 rounded text-xs text-red-300" data-testid="preview-error">
          {previewError}
        </div>
      )}

      {preview && (
        <div className="mt-4 p-4 bg-[var(--bg)] border border-[var(--border)] rounded-lg" data-testid="preview-result">
          {/* No-action warning */}
          <div className="mb-3 p-2 bg-blue-900/20 border border-blue-700 rounded text-xs text-blue-300" data-testid="preview-warning">
            ⚠ Preview only — no action has been requested or executed.
          </div>

          <div className="grid grid-cols-2 gap-x-4 gap-y-2 text-xs">
            {/* Action Type */}
            <div>
              <span className="text-[var(--text-muted)]">Action:</span>{' '}
              <span className="font-mono">{preview.action_type}</span>
            </div>

            {/* Target */}
            <div data-testid="preview-target">
              <span className="text-[var(--text-muted)]">Target:</span>{' '}
              <span>{preview.target.name} ({preview.target.node}:{preview.target.vmid})</span>
              <span className="text-[var(--text-muted)] ml-2">{preview.target.vm_type}</span>
            </div>

            {/* Risk Level */}
            <div>
              <span className="text-[var(--text-muted)]">Risk Level:</span>{' '}
              <span className={`font-semibold ${riskColor(preview.risk_level)}`} data-testid="preview-risk">
                {preview.risk_level}
              </span>
            </div>

            {/* Approval */}
            <div>
              <span className="text-[var(--text-muted)]">Approval Required:</span>{' '}
              <span data-testid="preview-approval">{preview.requires_approval ? 'Yes' : 'No'}</span>
            </div>

            {/* MFA */}
            <div>
              <span className="text-[var(--text-muted)]">MFA Required:</span>{' '}
              <span data-testid="preview-mfa">{preview.requires_mfa ? 'Yes' : 'No'}</span>
            </div>

            {/* Min Approvers */}
            <div>
              <span className="text-[var(--text-muted)]">Min Approvers:</span>{' '}
              <span data-testid="preview-min-approvers">{preview.min_approvers}</span>
            </div>

            {/* Mutation Window */}
            <div>
              <span className="text-[var(--text-muted)]">Mutation Window:</span>{' '}
              <span data-testid="preview-mutation-window">
                {preview.mutation_window_active ? '🟢 Active' : '🔴 Inactive'}
              </span>
              {preview.mutation_window_required && !preview.mutation_window_active && (
                <span className="text-red-400 ml-1">(required)</span>
              )}
            </div>

            {/* Feature Flag */}
            <div>
              <span className="text-[var(--text-muted)]">Feature Flag:</span>{' '}
              <span data-testid="preview-feature-flag">
                {preview.feature_flag_enabled ? '🟢 Enabled' : '🔴 Disabled'}
              </span>
            </div>

            {/* Currently Blocked? */}
            <div className="col-span-2 mt-2">
              <span className="text-[var(--text-muted)]">Execution would currently be:</span>{' '}
              <span className="font-semibold" data-testid="preview-blocked-status">
                {(!preview.feature_flag_enabled || (preview.mutation_window_required && !preview.mutation_window_active))
                  ? '🔒 Blocked'
                  : '✅ Allowed (after approval + MFA)'}
              </span>
            </div>
          </div>

          {/* Validation */}
          <div className="mt-3 text-xs text-[var(--text-muted)]" data-testid="preview-validation">
            Validation: {preview.validation.asset_valid ? '✓' : '✗'} asset{' '}
            {preview.validation.target_valid ? '✓' : '✗'} target{' '}
            {preview.validation.snapshot_name_valid ? '✓' : '✗'} snapshot{' '}
            {preview.validation.policy_valid ? '✓' : '✗'} policy
          </div>
        </div>
      )}

      {!canCreate && canExecute && (
        <p className="mt-2 text-xs text-[var(--text-muted)]">You have execute permission but not create permission.</p>
      )}
    </div>
  );
}
