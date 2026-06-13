import { useState } from 'react';
import { api, ApiError } from '../../api/client';
import { usePermissions } from '../../hooks/usePermissions';

interface AssetActionsProps {
  assetId: string;
  hostname?: string;
}

export default function AssetActions({ assetId, hostname }: AssetActionsProps) {
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [snapshotName, setSnapshotName] = useState('');
  const [busy, setBusy] = useState<string | null>(null);
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

  if (!canCreate && !hasPermission('assets.actions.read')) return null;

  return (
    <div className="mt-4 p-4 bg-[var(--card)] border border-[var(--border)] rounded-lg">
      <h3 className="text-sm font-semibold mb-3">Proxmox Actions {hostname ? `— ${hostname}` : ''}</h3>

      {error && <div className="mb-3 p-2 bg-red-900/30 border border-red-700 rounded text-xs text-red-300">{error}</div>}
      {success && <div className="mb-3 p-2 bg-green-900/20 border border-green-700 rounded text-xs text-green-300">{success}</div>}

      <div className="grid grid-cols-2 gap-2" data-testid="asset-action-buttons">
        {/* Start */}
        <button
          onClick={() => handleAction('start')}
          disabled={!canCreate || busy !== null}
          data-testid="btn-start"
          className="px-3 py-2 bg-green-900/30 border border-green-700 rounded text-sm text-green-300 hover:bg-green-900/50 disabled:opacity-50"
        >
          ▶ Start
          <span className="block text-xs text-[var(--text-muted)]">Medium risk • Approval required</span>
        </button>

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
        <button
          onClick={() => handleAction('shutdown')}
          disabled={!canCreate || busy !== null}
          data-testid="btn-shutdown"
          className="px-3 py-2 bg-orange-900/30 border border-orange-700 rounded text-sm text-orange-300 hover:bg-orange-900/50 disabled:opacity-50"
        >
          ⏹ Shutdown
          <span className="block text-xs text-[var(--text-muted)]">High risk • MFA required</span>
        </button>

        {/* Stop — elevated */}
        <button
          onClick={() => handleAction('stop')}
          disabled={!canCreate || busy !== null}
          data-testid="btn-stop"
          className="px-3 py-2 bg-red-900/40 border border-red-700 rounded text-sm text-red-300 hover:bg-red-900/60 disabled:opacity-50"
        >
          ⛔ Stop (Force)
          <span className="block text-xs text-red-400 font-medium" data-testid="stop-warning">
            Critical • 2 approvers + MFA required
          </span>
        </button>
      </div>

      {!canCreate && canExecute && (
        <p className="mt-2 text-xs text-[var(--text-muted)]">You have execute permission but not create permission.</p>
      )}
    </div>
  );
}
