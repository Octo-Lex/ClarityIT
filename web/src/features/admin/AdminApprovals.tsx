import { useState, useEffect, useCallback } from 'react';
import { api } from '../../api/client';
import { usePermissions } from '../../hooks/usePermissions';
import { useWebSocketInvalidation } from '../../hooks/useWebSocket';
import { useRefetch } from '../../hooks/useRefetch';
import PolicySimulationPanel from './PolicySimulationPanel';

function formatRemaining(seconds: number): string {
  if (seconds <= 0) return 'expired';
  if (seconds < 60) return `${seconds}s`;
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  if (m < 60) return `${m}m ${s}s`;
  const h = Math.floor(m / 60);
  return `${h}h ${m % 60}m`;
}

function ExpiryBadge({ remaining, isExpiring, status }: { remaining: number; isExpiring: boolean; status: string }) {
  if (status === 'expired') {
    return <span className="text-xs px-2 py-0.5 rounded bg-destructive/15 text-destructive font-medium" data-testid="expired-badge">EXPIRED</span>;
  }
  if (remaining <= 0) {
    return <span className="text-xs px-2 py-0.5 rounded bg-destructive/15 text-destructive font-medium" data-testid="expired-badge">EXPIRED</span>;
  }
  if (isExpiring || remaining < 300) {
    return <span className="text-xs px-2 py-0.5 rounded bg-warning/20 text-warning font-medium" data-testid="expiring-badge">⚠ EXPIRING ({formatRemaining(remaining)})</span>;
  }
  return <span className="text-xs px-2 py-0.5 rounded bg-success/15 text-success" data-testid="active-badge">{formatRemaining(remaining)} left</span>;
}

export default function AdminApprovals() {
  const [approvals, setApprovals] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [reason, setReason] = useState<Record<string, string>>({});
  const [statusFilter, setStatusFilter] = useState('pending');
  const [toast, setToast] = useState<string | null>(null);
  const { hasPermission, isPlatformOwner } = usePermissions();
  const { lastEvent: wsEvent } = useWebSocketInvalidation();
  const { bump } = useRefetch();

  const canApprove = hasPermission('approvals.approve');

  const load = useCallback(async () => {
    try {
      const data = await api.listApprovals(statusFilter);
      setApprovals(data || []);
    } catch (e) {
      setError('Failed to load approvals');
    }
    setLoading(false);
  }, [statusFilter]);

  useEffect(() => { load(); }, [load]);

  // Handle WS events for approval expiry notifications
  useEffect(() => {
    if (!wsEvent) return;
    const evt = wsEvent;
    if (evt.event_type === 'approval.expiring') {
      setToast(`⚠ Approval expiring soon: ${evt.action_type || 'action'}`);
      bump();
      setTimeout(() => setToast(null), 8000);
    } else if (evt.event_type === 'approval.expired') {
      setToast(`Approval expired: ${evt.action_type || 'action'}`);
      bump();
      setTimeout(() => setToast(null), 8000);
    }
  }, [wsEvent, bump]);

  // Auto-refresh remaining times every 15 seconds
  useEffect(() => {
    if (statusFilter !== 'pending') return;
    const interval = setInterval(() => {
      setApprovals(prev => prev.map(a => ({
        ...a,
        remaining_seconds: a.remaining_seconds > 0 ? a.remaining_seconds - 15 : 0,
      })));
    }, 15000);
    return () => clearInterval(interval);
  }, [statusFilter]);

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

  if (loading) return <div className="p-6 text-muted-foreground">Loading...</div>;

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-bold">Approvals</h1>

      {toast && (
        <div className="p-3 bg-warning/20 border border-warning/40 rounded text-sm text-warning" data-testid="approval-toast">
          {toast}
        </div>
      )}

      {error && <div className="p-3 bg-destructive/15 border border-destructive/40 rounded text-sm text-destructive">{error}</div>}

      {/* Status filter */}
      <div className="flex gap-2">
        {['pending', 'approved', 'rejected', 'expired', 'cancelled'].map(s => (
          <button
            key={s}
            onClick={() => setStatusFilter(s)}
            className={`px-3 py-1 rounded text-sm capitalize ${
              statusFilter === s
                ? 'bg-primary text-white'
                : 'bg-surface border border-border text-muted-foreground'
            }`}
          >
            {s}
          </button>
        ))}
      </div>

      {approvals.length === 0 ? (
        <p className="text-muted-foreground">No {statusFilter} approvals.</p>
      ) : (
        <div className="space-y-3">
          {approvals.map(a => (
            <div key={a.id} className="p-4 bg-surface border border-border rounded-lg" data-testid={`approval-${a.id}`}>
              <div className="flex items-center justify-between mb-2">
                <div className="flex items-center gap-2">
                  <span className="font-medium">{a.action_type}</span>
                  <ExpiryBadge
                    remaining={a.remaining_seconds || 0}
                    isExpiring={a.is_expiring || false}
                    status={a.status}
                  />
                </div>
                <span className={`text-xs px-2 py-0.5 rounded ${
                  a.risk_level === 'critical' ? 'bg-destructive/15 text-destructive' :
                  a.risk_level === 'high' ? 'bg-warning/20 text-warning' :
                  a.risk_level === 'medium' ? 'bg-warning/20 text-warning' :
                  'bg-success/15 text-success'
                }`}>{a.risk_level}</span>
              </div>
              <p className="text-sm text-muted-foreground mb-2">{a.description}</p>
              <p className="text-xs text-muted-foreground mb-3">
                Requested by: {a.requested_by?.slice(0, 8)}... • Created: {new Date(a.created_at).toLocaleString()}
                {a.expires_at && ` • Expires: ${new Date(a.expires_at).toLocaleString()}`}
              </p>

              {(a.risk_level === 'high' || a.risk_level === 'critical') && a.status === 'pending' && (
                <p className="text-xs text-warning mb-2" data-testid={`mfa-required-${a.id}`}>
                  ⚠ Recent MFA verification required to approve
                </p>
              )}

              {a.risk_level === 'critical' && a.status === 'pending' && (
                <p className="text-xs text-destructive mb-2" data-testid={`critical-warning-${a.id}`}>
                  ⚠ Critical action: 2 distinct approvers required
                </p>
              )}

              {canApprove && a.status === 'pending' && (
                <div className="flex gap-2 items-center">
                  <input
                    type="text"
                    placeholder="Reason (optional)"
                    value={reason[a.id] || ''}
                    onChange={e => setReason({ ...reason, [a.id]: e.target.value })}
                    className="flex-1 px-3 py-1.5 bg-background border border-border rounded text-sm"
                  />
                  <button
                    onClick={() => handleApprove(a.id)}
                    data-testid={`approve-btn-${a.id}`}
                    className="px-3 py-1.5 bg-success text-white rounded text-sm"
                  >
                    Approve
                  </button>
                  <button
                    onClick={() => handleReject(a.id)}
                    data-testid={`reject-btn-${a.id}`}
                    className="px-3 py-1.5 bg-destructive/15 border border-destructive/40 text-destructive rounded text-sm"
                  >
                    Reject
                  </button>
                </div>
              )}
            </div>
          ))}
        </div>
      )}

      {/* v1.2 Track 3: Approval Policy Simulation */}
      <PolicySimulationPanel isPlatformOwner={isPlatformOwner} />
    </div>
  );
}
