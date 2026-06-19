import { useState, useEffect, useCallback } from 'react';
import { api, ApiError } from '../../api/client';
import { usePermissions } from '../../hooks/usePermissions';
import EvidencePanel from '../remediation/EvidencePanel';
import OutcomePanel from '../shared/OutcomePanel';

interface RemediationPanelProps {
  incidentId?: string;
}

export default function RemediationPanel({ incidentId }: RemediationPanelProps) {
  const [proposals, setProposals] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [expanded, setExpanded] = useState<string | null>(null);
  const { hasPermission } = usePermissions();

  const canApprove = hasPermission('remediations.approve');
  const canExecute = hasPermission('remediations.execute');
  const canCancel = hasPermission('remediations.cancel');

  const load = useCallback(async () => {
    try {
      const all = await api.listRemediations();
      const filtered = incidentId ? (all || []).filter((p: any) => p.incident_id === incidentId) : (all || []);
      setProposals(filtered);
    } catch { /* ignore */ }
    setLoading(false);
  }, [incidentId]);

  useEffect(() => { load(); }, [load]);

  const handleApprove = async (id: string) => {
    setError('');
    try {
      await api.approveRemediation(id);
      await load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Approve failed');
    }
  };

  const handleExecute = async (id: string) => {
    setError('');
    try {
      await api.executeRemediation(id);
      await load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Execute failed');
    }
  };

  const handleCancel = async (id: string) => {
    setError('');
    try {
      await api.cancelRemediation(id);
      await load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Cancel failed');
    }
  };

  if (loading) return <div className="text-muted-foreground">Loading remediations...</div>;

  return (
    <div className="space-y-3">
      <h2 className="text-lg font-semibold">Remediation Proposals</h2>
      {error && <div className="p-2 bg-destructive/15 border border-destructive/40 rounded text-xs text-destructive">{error}</div>}

      {proposals.length === 0 ? (
        <p className="text-sm text-muted-foreground">No remediation proposals.</p>
      ) : (
        proposals.map(p => (
          <div key={p.id} className="p-3 bg-surface border border-border rounded" data-testid={`remediation-${p.id}`}>
            <div
              className="flex items-center justify-between cursor-pointer"
              onClick={() => setExpanded(expanded === p.id ? null : p.id)}
            >
              <div>
                <span className="font-medium text-sm">{p.title}</span>
                <span className={`ml-2 text-xs px-1.5 py-0.5 rounded ${
                  p.status === 'completed' ? 'bg-success/15 text-success' :
                  p.status === 'failed' ? 'bg-destructive/15 text-destructive' :
                  p.status === 'executing' ? 'bg-info/15 text-info' :
                  p.status === 'approved' ? 'bg-cyan-900/40 text-cyan-300' :
                  p.status === 'cancelled' ? 'bg-muted text-muted-foreground' :
                  'bg-warning/20 text-warning'
                }`} data-testid={`remediation-status-${p.id}`}>{p.status}</span>
              </div>
              <span className="text-xs text-muted-foreground">{p.source}</span>
            </div>

            {/* MFA indicator for high-risk */}
            {(p.risk_level === 'high' || p.risk_level === 'critical') && p.status === 'approved' && (
              <p className="mt-1 text-xs text-warning" data-testid={`remediation-mfa-${p.id}`}>
                ⚠ MFA required to execute this high-risk remediation
              </p>
            )}

            {expanded === p.id && (
              <div className="mt-3 space-y-2" data-testid={`remediation-detail-${p.id}`}>
                <p className="text-xs text-muted-foreground">{p.description}</p>

                {/* v1.2 Track 1: Evidence Panel */}
                <EvidencePanel recommendationId={p.id} />

                {/* v1.2 Track 5: Outcome Panel */}
                <OutcomePanel sourceType="remediation" sourceId={p.id} sourceStatus={p.status} />

                {/* Steps */}
                <div className="space-y-1" data-testid={`remediation-steps-${p.id}`}>
                  <RemediationSteps proposalId={p.id} />
                </div>

                {/* Actions */}
                <div className="flex gap-2 mt-2">
                  {p.status === 'proposed' || p.status === 'draft' ? (
                    canApprove && (
                      <button
                        onClick={() => handleApprove(p.id)}
                        data-testid={`remediation-approve-${p.id}`}
                        className="px-3 py-1 bg-success text-white rounded text-xs"
                      >
                        Approve
                      </button>
                    )
                  ) : null}
                  {p.status === 'approved' && canExecute && (
                    <button
                      onClick={() => handleExecute(p.id)}
                      data-testid={`remediation-execute-${p.id}`}
                      className="px-3 py-1 bg-primary text-white rounded text-xs"
                    >
                      Execute
                    </button>
                  )}
                  {['draft', 'proposed', 'approved'].includes(p.status) && canCancel && (
                    <button
                      onClick={() => handleCancel(p.id)}
                      data-testid={`remediation-cancel-${p.id}`}
                      className="px-3 py-1 bg-destructive/15 border border-destructive/40 text-destructive rounded text-xs"
                    >
                      Cancel
                    </button>
                  )}
                </div>
              </div>
            )}
          </div>
        ))
      )}
    </div>
  );
}

// Sub-component to lazy-load steps
function RemediationSteps({ proposalId }: { proposalId: string }) {
  const [detail, setDetail] = useState<any>(null);
  const [loaded, setLoaded] = useState(false);

  useEffect(() => {
    api.getRemediation(proposalId).then(d => { setDetail(d); setLoaded(true); }).catch(() => setLoaded(true));
  }, [proposalId]);

  if (!loaded) return <p className="text-xs text-muted-foreground">Loading steps...</p>;
  if (!detail?.steps?.length) return <p className="text-xs text-muted-foreground">No steps.</p>;

  return (
    <>
      <p className="text-xs font-medium text-muted-foreground">Steps:</p>
      {(detail.steps as any[]).map((s, i) => (
        <div key={s.id || i} className="flex items-center gap-2 text-xs p-1.5 bg-background rounded">
          <span className="text-muted-foreground">#{s.step_order}</span>
          <span className="font-mono">{s.tool_name}</span>
          <span className={`px-1 rounded ${
            s.status === 'succeeded' ? 'text-success' :
            s.status === 'failed' ? 'text-destructive' :
            s.status === 'executing' ? 'text-info' :
            'text-muted-foreground'
          }`}>{s.status}</span>
          {s.error_message && (
            <span className="text-destructive" data-testid={`step-error-${s.id}`}>
              {s.error_message}
            </span>
          )}
        </div>
      ))}
    </>
  );
}
