import { useState, useEffect, useCallback } from 'react';
import { api } from '../../api/client';
import { usePermissions } from '../../hooks/usePermissions';

export default function AdminApprovals() {
  const [approvals, setApprovals] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [reason, setReason] = useState<Record<string, string>>({});
  const { hasPermission } = usePermissions();

  const canApprove = hasPermission('approvals.approve');

  const load = useCallback(async () => {
    try {
      const data = await api.listApprovals('pending');
      setApprovals(data || []);
    } catch (e) {
      setError('Failed to load approvals');
    }
    setLoading(false);
  }, []);

  useEffect(() => { load(); }, [load]);

  const handleApprove = async (id: string) => {
    setError('');
    try {
      await api.approveApproval(id, reason[id] || '');
      await load();
    } catch (e: any) {
      setError(e.message || 'Approve failed');
    }
  };

  const handleReject = async (id: string) => {
    setError('');
    try {
      await api.rejectApproval(id, reason[id] || '');
      await load();
    } catch (e: any) {
      setError(e.message || 'Reject failed');
    }
  };

  if (loading) return <div className="p-6 text-[var(--text-muted)]">Loading...</div>;

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-bold">Approvals</h1>
      {error && <div className="p-3 bg-red-900/30 border border-red-700 rounded text-sm text-red-300">{error}</div>}

      {approvals.length === 0 ? (
        <p className="text-[var(--text-muted)]">No pending approvals.</p>
      ) : (
        <div className="space-y-3">
          {approvals.map(a => (
            <div key={a.id} className="p-4 bg-[var(--card)] border border-[var(--border)] rounded-lg" data-testid={`approval-${a.id}`}>
              <div className="flex items-center justify-between mb-2">
                <span className="font-medium">{a.action_type}</span>
                <span className={`text-xs px-2 py-0.5 rounded ${
                  a.risk_level === 'critical' ? 'bg-red-900/40 text-red-300' :
                  a.risk_level === 'high' ? 'bg-orange-900/40 text-orange-300' :
                  a.risk_level === 'medium' ? 'bg-yellow-900/40 text-yellow-300' :
                  'bg-green-900/40 text-green-300'
                }`}>{a.risk_level}</span>
              </div>
              <p className="text-sm text-[var(--text-muted)] mb-2">{a.description}</p>
              <p className="text-xs text-[var(--text-muted)] mb-3">
                Requested by: {a.requested_by?.slice(0, 8)}... • Created: {new Date(a.created_at).toLocaleString()}
              </p>

              {/* MFA requirement indicator */}
              {(a.risk_level === 'high' || a.risk_level === 'critical') && (
                <p className="text-xs text-orange-400 mb-2" data-testid={`mfa-required-${a.id}`}>
                  ⚠ Recent MFA verification required to approve
                </p>
              )}

              {/* Critical elevated approval indicator */}
              {a.risk_level === 'critical' && (
                <p className="text-xs text-red-400 mb-2" data-testid={`critical-warning-${a.id}`}>
                  ⚠ Critical action: 2 distinct approvers required
                </p>
              )}

              {canApprove && (
                <div className="flex gap-2 items-center">
                  <input
                    type="text"
                    placeholder="Reason (optional)"
                    value={reason[a.id] || ''}
                    onChange={e => setReason({ ...reason, [a.id]: e.target.value })}
                    className="flex-1 px-3 py-1.5 bg-[var(--bg)] border border-[var(--border)] rounded text-sm"
                  />
                  <button
                    onClick={() => handleApprove(a.id)}
                    data-testid={`approve-btn-${a.id}`}
                    className="px-3 py-1.5 bg-[var(--success)] text-white rounded text-sm"
                  >
                    Approve
                  </button>
                  <button
                    onClick={() => handleReject(a.id)}
                    data-testid={`reject-btn-${a.id}`}
                    className="px-3 py-1.5 bg-red-900/40 border border-red-700 text-red-300 rounded text-sm"
                  >
                    Reject
                  </button>
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
