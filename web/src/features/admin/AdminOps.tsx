import { useEffect, useState, useCallback } from 'react';
import { api } from '../../api/client';

interface Summary {
  outbox_pending: number;
  dead_letters: number;
  agent_runs_pending: number;
  agent_runs_running: number;
  webhook_rejections_24h: number;
  agent_blocks_24h: number;
  security_events_24h: number;
  integration_keys_active: number;
  integration_keys_rotation_required: number;
  total_users: number;
  total_teams: number;
}

interface HealthData {
  postgres: { status: string; latency?: string };
  nats: { status: string; latency?: string };
  redis: { status: string; latency?: string };
  minio: { status: string; latency?: string };
  outbox: { pending: number; dead_letter: number };
  uptime: string;
}

interface Worker {
  name: string;
  status: string;
  last_seen?: string | null;
}

const statusColor = (status: string): string => {
  switch (status) {
    case 'up':
    case 'active':
      return 'text-[var(--success)]';
    case 'degraded':
    case 'stale':
      return 'text-[var(--warning)]';
    case 'down':
      return 'text-[var(--danger)]';
    default:
      return 'text-[var(--text-muted)]';
  }
};

const statusIcon = (status: string): string => {
  switch (status) {
    case 'up':
    case 'active':
      return '●';
    case 'degraded':
    case 'stale':
      return '◆';
    case 'down':
      return '✕';
    default:
      return '○';
  }
};

function StatCard({ label, value, warning }: { label: string; value: number | string; warning?: boolean }) {
  return (
    <div className="card p-4">
      <div className="text-2xl font-bold" style={{ color: warning ? 'var(--warning)' : 'inherit' }}>{value}</div>
      <div className="text-xs text-[var(--text-muted)] mt-1">{label}</div>
    </div>
  );
}

export default function AdminOps() {
  const [summary, setSummary] = useState<Summary | null>(null);
  const [health, setHealth] = useState<HealthData | null>(null);
  const [workers, setWorkers] = useState<Worker[]>([]);
  const [deadLetters, setDeadLetters] = useState<any[]>([]);
  const [webhookEvents, setWebhookEvents] = useState<any[]>([]);
  const [agentBlocks, setAgentBlocks] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const load = useCallback(async () => {
    try {
      const [s, h, w, dl, we, ab] = await Promise.all([
        api.opsSummary(),
        api.deepHealth().catch(() => null),
        api.opsWorkers().catch(() => []),
        api.opsDeadLetters().catch(() => []),
        api.opsWebhookRejections().catch(() => []),
        api.opsAgentBlocks().catch(() => []),
      ]);
      setSummary(s as Summary);
      setHealth(h as HealthData);
      setWorkers(w);
      setDeadLetters(dl);
      setWebhookEvents(we);
      setAgentBlocks(ab);
      setError('');
    } catch (e: any) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
    const interval = setInterval(load, 15000); // auto-refresh every 15s
    return () => clearInterval(interval);
  }, [load]);

  if (loading && !summary) return <div className="text-[var(--text-muted)]">Loading ops dashboard...</div>;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Operations Dashboard</h1>
        <button onClick={load} className="text-sm px-3 py-1 rounded bg-[var(--primary)] text-white">↻ Refresh</button>
      </div>

      {error && <div className="card p-3 text-[var(--danger)]">{error}</div>}

      {/* System Health */}
      {health && (
        <div>
          <h2 className="text-lg font-semibold mb-2">System Health</h2>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
            {(['postgres', 'nats', 'redis', 'minio'] as const).map(dep => {
              const d = health[dep];
              return (
                <div key={dep} className="card p-3">
                  <div className="text-sm font-medium capitalize">{dep}</div>
                  <div className={`text-lg font-bold ${statusColor(d?.status || 'unknown')}`}>
                    {statusIcon(d?.status || 'unknown')} {d?.status || 'unknown'}
                  </div>
                  {d?.latency && <div className="text-xs text-[var(--text-muted)]">{d.latency}</div>}
                </div>
              );
            })}
          </div>
          {health.uptime && <div className="text-xs text-[var(--text-muted)] mt-2">Uptime: {health.uptime}</div>}
        </div>
      )}

      {/* Summary Stats */}
      {summary && (
        <div>
          <h2 className="text-lg font-semibold mb-2">Summary (24h)</h2>
          <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-3">
            <StatCard label="Outbox Pending" value={summary.outbox_pending} warning={summary.outbox_pending > 10} />
            <StatCard label="Dead Letters" value={summary.dead_letters} warning={summary.dead_letters > 0} />
            <StatCard label="Agent Runs Pending" value={summary.agent_runs_pending} />
            <StatCard label="Webhook Rejections" value={summary.webhook_rejections_24h} warning={summary.webhook_rejections_24h > 0} />
            <StatCard label="Agent Blocks" value={summary.agent_blocks_24h} />
            <StatCard label="Security Events" value={summary.security_events_24h} warning={summary.security_events_24h > 5} />
            <StatCard label="Active Keys" value={summary.integration_keys_active} />
            <StatCard label="Keys Needing Rotation" value={summary.integration_keys_rotation_required} warning={summary.integration_keys_rotation_required > 0} />
            <StatCard label="Users" value={summary.total_users} />
            <StatCard label="Teams" value={summary.total_teams} />
          </div>
        </div>
      )}

      {/* Workers */}
      <div>
        <h2 className="text-lg font-semibold mb-2">Worker Status</h2>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
          {workers.map(w => (
            <div key={w.name} className="card p-3">
              <div className="flex justify-between items-center">
                <span className="text-sm font-medium">{w.name}</span>
                <span className={`text-sm font-bold ${statusColor(w.status)}`}>
                  {statusIcon(w.status)} {w.status}
                </span>
              </div>
              {w.last_seen && (
                <div className="text-xs text-[var(--text-muted)] mt-1">
                  Last seen: {new Date(w.last_seen).toLocaleString()}
                </div>
              )}
            </div>
          ))}
        </div>
      </div>

      {/* Dead Letters */}
      <div>
        <h2 className="text-lg font-semibold mb-2">Dead Letters ({deadLetters.length})</h2>
        <div className="card overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left text-[var(--text-muted)] border-b border-[var(--border)]">
                <th className="pb-2 pr-4">Event Type</th>
                <th className="pb-2 pr-4">Aggregate</th>
                <th className="pb-2 pr-4">Attempts</th>
                <th className="pb-2 pr-4">Error</th>
                <th className="pb-2">Dead At</th>
              </tr>
            </thead>
            <tbody>
              {deadLetters.map((d) => (
                <tr key={d.id} className="border-b border-[var(--border)]">
                  <td className="py-2 pr-4 font-mono text-xs">{d.event_type}</td>
                  <td className="py-2 pr-4 text-xs">{d.aggregate_type}/{d.aggregate_id?.slice(0, 8)}…</td>
                  <td className="py-2 pr-4">{d.attempts}</td>
                  <td className="py-2 pr-4 text-xs text-[var(--danger)] max-w-xs truncate">{d.error_message || '—'}</td>
                  <td className="py-2 text-xs text-[var(--text-muted)]">{d.dead_lettered_at ? new Date(d.dead_lettered_at).toLocaleString() : '—'}</td>
                </tr>
              ))}
              {!deadLetters.length && (
                <tr><td colSpan={5} className="py-4 text-center text-[var(--text-muted)]">No dead letters</td></tr>
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* Webhook Rejections */}
      <div>
        <h2 className="text-lg font-semibold mb-2">Webhook Events (24h)</h2>
        <div className="card overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left text-[var(--text-muted)] border-b border-[var(--border)]">
                <th className="pb-2 pr-4">Action</th>
                <th className="pb-2 pr-4">Summary</th>
                <th className="pb-2">Time</th>
              </tr>
            </thead>
            <tbody>
              {webhookEvents.map((e) => (
                <tr key={e.event_id} className="border-b border-[var(--border)]">
                  <td className="py-2 pr-4 font-mono text-xs">{e.action}</td>
                  <td className="py-2 pr-4 text-xs">{e.summary || '—'}</td>
                  <td className="py-2 text-xs text-[var(--text-muted)]">{new Date(e.created_at).toLocaleString()}</td>
                </tr>
              ))}
              {!webhookEvents.length && (
                <tr><td colSpan={3} className="py-4 text-center text-[var(--text-muted)]">No webhook events in last 24h</td></tr>
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* Agent Blocked Actions */}
      <div>
        <h2 className="text-lg font-semibold mb-2">Agent Blocked/Denied Actions</h2>
        <div className="card overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left text-[var(--text-muted)] border-b border-[var(--border)]">
                <th className="pb-2 pr-4">Status</th>
                <th className="pb-2 pr-4">Tool</th>
                <th className="pb-2 pr-4">Result</th>
                <th className="pb-2">Time</th>
              </tr>
            </thead>
            <tbody>
              {agentBlocks.map((b) => (
                <tr key={b.effect_id} className="border-b border-[var(--border)]">
                  <td className={`py-2 pr-4 font-bold ${statusColor('down')}`}>{b.status}</td>
                  <td className="py-2 pr-4 font-mono text-xs">{b.tool_name}</td>
                  <td className="py-2 pr-4 text-xs max-w-md truncate">{b.result || '—'}</td>
                  <td className="py-2 text-xs text-[var(--text-muted)]">{new Date(b.created_at).toLocaleString()}</td>
                </tr>
              ))}
              {!agentBlocks.length && (
                <tr><td colSpan={4} className="py-4 text-center text-[var(--text-muted)]">No blocked/denied agent actions</td></tr>
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
