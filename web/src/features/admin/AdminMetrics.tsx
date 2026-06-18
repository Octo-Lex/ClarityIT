import { useState, useEffect } from 'react';
import { api, ApiError, type AdminMetrics } from '../../api/client';
import { usePermissions } from '../../hooks/usePermissions';

function MetricCard({ label, value, color, section }: { label: string; value: string | number; color?: string; section: string }) {
  return (
    <div className="p-3 bg-[var(--bg)] border border-[var(--border)] rounded-lg">
      <div className="text-xs text-[var(--text-muted)]">{label}</div>
      <div className={`text-lg font-semibold ${color || ''}`} data-testid={`${section}-${label.toLowerCase().replace(/[^a-z0-9]+/g, '-')}`}>
        {value}
      </div>
    </div>
  );
}

function SectionCard({ title, children, testId }: { title: string; children: React.ReactNode; testId: string }) {
  return (
    <div className="p-4 bg-[var(--card)] border border-[var(--border)] rounded-lg" data-testid={testId}>
      <h3 className="text-sm font-semibold mb-3">{title}</h3>
      <div className="grid grid-cols-3 gap-2">
        {children}
      </div>
    </div>
  );
}

export default function AdminMetrics() {
  const [metrics, setMetrics] = useState<AdminMetrics | null>(null);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);
  const { isPlatformOwner, hasPermission } = usePermissions();

  const canView = isPlatformOwner || hasPermission('ops.metrics.read');

  useEffect(() => {
    if (!canView) {
      setLoading(false);
      return;
    }
    let active = true;
    api.getMetrics()
      .then((data) => { if (active) { setMetrics(data); setLoading(false); } })
      .catch((e: unknown) => {
        if (active) {
          setError(e instanceof ApiError ? e.message : 'Failed to load metrics');
          setLoading(false);
        }
      });
    return () => { active = false; };
  }, [canView]);

  if (!canView) {
    return (
      <div className="p-8 text-center text-[var(--text-muted)]" data-testid="metrics-permission-denied">
        You need platform admin permissions to view operational metrics.
      </div>
    );
  }

  if (loading) {
    return <div className="p-8 text-center text-[var(--text-muted)]">Loading metrics...</div>;
  }

  if (error) {
    return (
      <div className="p-4 bg-red-900/30 border border-red-700 rounded text-red-300" data-testid="metrics-error">
        {error}
      </div>
    );
  }

  if (!metrics) return null;

  const formatTime = (seconds: number) => {
    if (seconds < 60) return `${seconds.toFixed(1)}s`;
    return `${(seconds / 60).toFixed(1)}m`;
  };

  return (
    <div className="space-y-4" data-testid="metrics-page">
      <h2 className="text-lg font-semibold">Operational Metrics</h2>

      {/* Approvals */}
      <SectionCard title="Approvals" testId="metrics-approvals">
        <MetricCard section="approvals" label="Pending" value={metrics.approvals.pending} />
        <MetricCard section="approvals" label="Approved" value={metrics.approvals.approved} color="text-green-400" />
        <MetricCard section="approvals" label="Rejected" value={metrics.approvals.rejected} color="text-red-400" />
        <MetricCard section="approvals" label="Expired" value={metrics.approvals.expired} color="text-orange-400" />
        <MetricCard section="approvals" label="Executed" value={metrics.approvals.executed} />
        <MetricCard section="approvals" label="Failed" value={metrics.approvals.failed} color="text-red-400" />
        <div className="col-span-3 mt-2 text-xs text-[var(--text-muted)]" data-testid="metrics-avg-decision">
          Avg time to decision: <span className="text-[var(--text)] font-semibold">{formatTime(metrics.approvals.avg_time_to_decision_seconds)}</span>
        </div>
      </SectionCard>

      {/* Remediations */}
      <SectionCard title="Remediations" testId="metrics-remediations">
        <MetricCard section="remediations" label="Draft" value={metrics.remediations.draft} />
        <MetricCard section="remediations" label="Proposed" value={metrics.remediations.proposed} />
        <MetricCard section="remediations" label="Approved" value={metrics.remediations.approved} />
        <MetricCard section="remediations" label="Executing" value={metrics.remediations.executing} color="text-blue-400" />
        <MetricCard section="remediations" label="Completed" value={metrics.remediations.completed} color="text-green-400" />
        <MetricCard section="remediations" label="Failed" value={metrics.remediations.failed} color="text-red-400" />
        <MetricCard section="remediations" label="Cancelled" value={metrics.remediations.cancelled} />
      </SectionCard>

      {/* Asset Actions */}
      <SectionCard title="Asset Actions" testId="metrics-asset-actions">
        <MetricCard section="assets" label="Pending" value={metrics.asset_actions.by_status.pending || 0} />
        <MetricCard section="assets" label="Executing" value={metrics.asset_actions.by_status.executing || 0} color="text-blue-400" />
        <MetricCard section="assets" label="Succeeded" value={metrics.asset_actions.by_status.succeeded || 0} color="text-green-400" />
        <MetricCard section="assets" label="Failed" value={metrics.asset_actions.by_status.failed || 0} color="text-red-400" />
        <MetricCard section="assets" label="Start" value={metrics.asset_actions.by_type['proxmox.start'] || 0} />
        <MetricCard section="assets" label="Shutdown" value={metrics.asset_actions.by_type['proxmox.shutdown'] || 0} />
        <MetricCard section="assets" label="Stop" value={metrics.asset_actions.by_type['proxmox.stop'] || 0} />
        <MetricCard section="assets" label="Snapshot" value={metrics.asset_actions.by_type['proxmox.snapshot'] || 0} />
        <div className="col-span-3 mt-2 text-xs text-[var(--text-muted)]" data-testid="metrics-success-rate">
          Success rate: <span className="text-[var(--text)] font-semibold">{metrics.asset_actions.success_rate_percent.toFixed(1)}%</span>
          <span className="ml-2 text-[var(--text-muted)]">(succeeded + failed only)</span>
        </div>
      </SectionCard>

      {/* Agents */}
      <SectionCard title="Agent Runs" testId="metrics-agents">
        <MetricCard section="agents" label="Pending" value={metrics.agents.runs_pending} />
        <MetricCard section="agents" label="Running" value={metrics.agents.runs_running} color="text-blue-400" />
        <MetricCard section="agents" label="Completed" value={metrics.agents.runs_completed} color="text-green-400" />
        <MetricCard section="agents" label="Failed" value={metrics.agents.runs_failed} color="text-red-400" />
        <div className="col-span-3 mt-2 text-xs text-[var(--text-muted)]" data-testid="metrics-avg-reasoning">
          Avg reasoning time: <span className="text-[var(--text)] font-semibold">{formatTime(metrics.agents.avg_reasoning_time_seconds)}</span>
        </div>
      </SectionCard>
    </div>
  );
}
