import { useState } from 'react';
import { api, ApiError } from '../../api/client';
import { usePermissions } from '../../hooks/usePermissions';

// v1.2 Track 4: Risk Score Display Component
function RiskScoreDisplay({ riskScore }: { riskScore: any }) {
  const score = riskScore.score ?? 0;
  const level = riskScore.level ?? 'unknown';
  const topFactors = riskScore.top_factors ?? [];

  const levelColor: Record<string, string> = {
    low: 'text-success',
    medium: 'text-warning',
    high: 'text-warning',
    critical: 'text-destructive',
    unknown: 'text-muted-foreground',
  };

  const levelBg: Record<string, string> = {
    low: 'bg-success/15 border-success/40',
    medium: 'bg-warning/20 border-warning/40',
    high: 'bg-warning/15 border-warning/40',
    critical: 'bg-destructive/15 border-destructive/40',
    unknown: 'bg-muted/50 border-border',
  };

  return (
    <div className="mt-4 pt-3 border-t border-border" data-testid="risk-score-section">
      <div className="flex items-center gap-3 mb-2">
        <h4 className="text-sm font-semibold">Change-Risk Score</h4>
        <span
          className={`px-3 py-1 rounded border text-sm font-bold ${levelBg[level] || levelBg.unknown} ${levelColor[level] || levelColor.unknown}`}
          data-testid="risk-score-badge"
        >
          {score} · {level.toUpperCase()}
        </span>
      </div>

      {/* Advisory-only warning */}
      <div className="mb-2 text-xs text-muted-foreground" data-testid="risk-score-advisory">
        ⚠ Risk score is advisory only. Approval, MFA, policy, and mutation-window controls still apply.
      </div>

      {/* Top factors */}
      {topFactors.length > 0 && (
        <div className="mb-2" data-testid="risk-score-top-factors">
          <span className="text-xs text-muted-foreground">Top factors: </span>
          {topFactors.map((f: string, i: number) => (
            <span key={f} className="text-xs inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-muted text-muted-foreground mr-1" data-testid={`risk-score-factor-${f}`}>
              {f.replace(/_/g, ' ')}
            </span>
          ))}
        </div>
      )}
    </div>
  );
}

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
  risk_score?: number;
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
      case 'medium': return 'text-warning';
      case 'high': return 'text-warning';
      case 'critical': return 'text-destructive';
      default: return 'text-muted-foreground';
    }
  };

  if (!canCreate && !hasPermission('assets.actions.read')) return null;

  return (
    <div className="mt-4 p-4 bg-surface border border-border rounded-lg" data-testid="asset-actions-container">
      <h3 className="text-sm font-semibold mb-3">Proxmox Actions {hostname ? `— ${hostname}` : ''}</h3>

      {error && <div className="mb-3 p-2 bg-destructive/15 border border-destructive/40 rounded text-xs text-destructive">{error}</div>}
      {success && <div className="mb-3 p-2 bg-success/15 border border-success/40 rounded text-xs text-success">{success}</div>}

      <div className="grid grid-cols-2 gap-2" data-testid="asset-action-buttons">
        {/* Start */}
        <div>
          <button
            onClick={() => handleAction('start')}
            disabled={!canCreate || busy !== null}
            data-testid="btn-start"
            className="w-full px-3 py-2 bg-success/15 border border-success/40 rounded text-sm text-success hover:bg-success/15 disabled:opacity-50"
          >
            ▶ Start
            <span className="block text-xs text-muted-foreground">Medium risk • Approval required</span>
          </button>
          <button
            onClick={() => handlePreview('start')}
            disabled={busy !== null}
            data-testid="btn-preview-start"
            className="w-full mt-1 px-2 py-1 text-xs text-muted-foreground hover:text-foreground underline"
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
            className="w-full px-3 py-2 bg-info/10 border border-info/40 rounded text-sm text-info hover:bg-info/20 disabled:opacity-50"
          >
            📸 Snapshot
            <span className="block text-xs text-muted-foreground">Medium risk • Approval required</span>
          </button>
          <button
            onClick={() => handlePreview('snapshot')}
            disabled={busy !== null}
            data-testid="btn-preview-snapshot"
            className="w-full mt-1 px-2 py-1 text-xs text-muted-foreground hover:text-foreground underline"
          >
            Preview
          </button>
          <input
            type="text"
            placeholder="snapshot name (optional)"
            value={snapshotName}
            onChange={e => setSnapshotName(e.target.value)}
            maxLength={40}
            className="w-full mt-1 px-2 py-1 bg-background border border-border rounded text-xs"
          />
        </div>

        {/* Shutdown */}
        <div>
          <button
            onClick={() => handleAction('shutdown')}
            disabled={!canCreate || busy !== null}
            data-testid="btn-shutdown"
            className="w-full px-3 py-2 bg-warning/15 border border-warning/40 rounded text-sm text-warning hover:bg-warning/15 disabled:opacity-50"
          >
            ⏹ Shutdown
            <span className="block text-xs text-muted-foreground">High risk • MFA required</span>
          </button>
          <button
            onClick={() => handlePreview('shutdown')}
            disabled={busy !== null}
            data-testid="btn-preview-shutdown"
            className="w-full mt-1 px-2 py-1 text-xs text-muted-foreground hover:text-foreground underline"
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
            className="w-full px-3 py-2 bg-destructive/15 border border-destructive/40 rounded text-sm text-destructive hover:bg-destructive/15 disabled:opacity-50"
          >
            ⛔ Stop (Force)
            <span className="block text-xs text-destructive font-medium" data-testid="stop-warning">
              Critical • 2 approvers + MFA required
            </span>
          </button>
          <button
            onClick={() => handlePreview('stop')}
            disabled={busy !== null}
            data-testid="btn-preview-stop"
            className="w-full mt-1 px-2 py-1 text-xs text-muted-foreground hover:text-foreground underline"
          >
            Preview
          </button>
        </div>
      </div>

      {/* Preview Result */}
      {previewError && (
        <div className="mt-3 p-2 bg-destructive/15 border border-destructive/40 rounded text-xs text-destructive" data-testid="preview-error">
          {previewError}
        </div>
      )}

      {preview && (
        <div className="mt-4 p-4 bg-background border border-border rounded-lg" data-testid="preview-result">
          {/* No-action warning */}
          <div className="mb-3 p-2 bg-info/10 border border-info/40 rounded text-xs text-info" data-testid="preview-warning">
            ⚠ Preview only — no action has been requested or executed.
          </div>

          <div className="grid grid-cols-2 gap-x-4 gap-y-2 text-xs">
            {/* Action Type */}
            <div>
              <span className="text-muted-foreground">Action:</span>{' '}
              <span className="font-mono">{preview.action_type}</span>
            </div>

            {/* Target */}
            <div data-testid="preview-target">
              <span className="text-muted-foreground">Target:</span>{' '}
              <span>{preview.target.name} ({preview.target.node}:{preview.target.vmid})</span>
              <span className="text-muted-foreground ml-2">{preview.target.vm_type}</span>
            </div>

            {/* Risk Level */}
            <div>
              <span className="text-muted-foreground">Risk Level:</span>{' '}
              <span className={`font-semibold ${riskColor(preview.risk_level)}`} data-testid="preview-risk">
                {preview.risk_level}
              </span>
            </div>

            {/* Approval */}
            <div>
              <span className="text-muted-foreground">Approval Required:</span>{' '}
              <span data-testid="preview-approval">{preview.requires_approval ? 'Yes' : 'No'}</span>
            </div>

            {/* MFA */}
            <div>
              <span className="text-muted-foreground">MFA Required:</span>{' '}
              <span data-testid="preview-mfa">{preview.requires_mfa ? 'Yes' : 'No'}</span>
            </div>

            {/* Min Approvers */}
            <div>
              <span className="text-muted-foreground">Min Approvers:</span>{' '}
              <span data-testid="preview-min-approvers">{preview.min_approvers}</span>
            </div>

            {/* Mutation Window */}
            <div>
              <span className="text-muted-foreground">Mutation Window:</span>{' '}
              <span data-testid="preview-mutation-window">
                {preview.mutation_window_active ? '🟢 Active' : '🔴 Inactive'}
              </span>
              {preview.mutation_window_required && !preview.mutation_window_active && (
                <span className="text-destructive ml-1">(required)</span>
              )}
            </div>

            {/* Feature Flag */}
            <div>
              <span className="text-muted-foreground">Feature Flag:</span>{' '}
              <span data-testid="preview-feature-flag">
                {preview.feature_flag_enabled ? '🟢 Enabled' : '🔴 Disabled'}
              </span>
            </div>

            {/* Currently Blocked? */}
            <div className="col-span-2 mt-2">
              <span className="text-muted-foreground">Execution would currently be:</span>{' '}
              <span className="font-semibold" data-testid="preview-blocked-status">
                {(!preview.feature_flag_enabled || (preview.mutation_window_required && !preview.mutation_window_active))
                  ? '🔒 Blocked'
                  : '✅ Allowed (after approval + MFA)'}
              </span>
            </div>
          </div>

          {/* Validation */}
          <div className="mt-3 text-xs text-muted-foreground" data-testid="preview-validation">
            Validation: {preview.validation.asset_valid ? '✓' : '✗'} asset{' '}
            {preview.validation.target_valid ? '✓' : '✗'} target{' '}
            {preview.validation.snapshot_name_valid ? '✓' : '✗'} snapshot{' '}
            {preview.validation.policy_valid ? '✓' : '✗'} policy
          </div>

          {/* v1.2 Track 4: Risk Score */}
          {preview.risk_score && (
            <RiskScoreDisplay riskScore={preview.risk_score} />
          )}
        </div>
      )}

      {!canCreate && canExecute && (
        <p className="mt-2 text-xs text-muted-foreground">You have execute permission but not create permission.</p>
      )}
    </div>
  );
}
