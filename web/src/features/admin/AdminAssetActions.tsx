import { useState, useEffect, useCallback } from 'react';
import { api } from '../../api/client';

export default function AdminAssetActions() {
  const [actions, setActions] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState('');

  const load = useCallback(async () => {
    try {
      const data = await api.listAssetActions(filter || undefined);
      setActions(data || []);
    } catch { /* ignore */ }
    setLoading(false);
  }, [filter]);

  useEffect(() => { load(); }, [load]);

  if (loading) return <div className="p-6 text-muted-foreground">Loading...</div>;

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Asset Actions</h1>
        <select
          value={filter}
          onChange={e => setFilter(e.target.value)}
          className="px-3 py-1.5 bg-surface border border-border rounded text-sm"
        >
          <option value="">All</option>
          <option value="pending">Pending</option>
          <option value="approved">Approved</option>
          <option value="executing">Executing</option>
          <option value="succeeded">Succeeded</option>
          <option value="failed">Failed</option>
          <option value="cancelled">Cancelled</option>
        </select>
      </div>

      {actions.length === 0 ? (
        <p className="text-muted-foreground">No asset actions found.</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left border-b border-border">
                <th className="py-2 px-3">Action</th>
                <th className="py-2 px-3">Status</th>
                <th className="py-2 px-3">Task ID</th>
                <th className="py-2 px-3">Error</th>
                <th className="py-2 px-3">Created</th>
              </tr>
            </thead>
            <tbody>
              {actions.map(a => (
                <tr key={a.id} className="border-b border-border/50" data-testid={`action-row-${a.id}`}>
                  <td className="py-2 px-3 font-mono text-xs">{a.action_type}</td>
                  <td className="py-2 px-3">
                    <span className={`text-xs px-2 py-0.5 rounded ${
                      a.status === 'succeeded' ? 'bg-success/15 text-success' :
                      a.status === 'failed' ? 'bg-destructive/15 text-destructive' :
                      a.status === 'executing' ? 'bg-info/15 text-info' :
                      a.status === 'cancelled' ? 'bg-muted text-muted-foreground' :
                      'bg-warning/20 text-warning'
                    }`}>{a.status}</span>
                  </td>
                  <td className="py-2 px-3 font-mono text-xs text-muted-foreground" data-testid={`task-id-${a.id}`}>
                    {a.proxmox_task_id || '—'}
                  </td>
                  <td className="py-2 px-3 text-xs text-destructive max-w-xs truncate" data-testid={`error-${a.id}`}>
                    {a.error_message || '—'}
                  </td>
                  <td className="py-2 px-3 text-xs text-muted-foreground">
                    {a.created_at ? new Date(a.created_at).toLocaleString() : '—'}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
