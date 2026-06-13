import { useEffect, useState } from 'react';
import { api, type WorkItem } from '../../api/client';
import { useAuth } from '../../auth/context';
import { useNavigate } from 'react-router-dom';

export default function QueuePage() {
  const { activeTeamId, hasPermission } = useAuth();
  const nav = useNavigate();
  const [items, setItems] = useState<WorkItem[]>([]);
  const [status, setStatus] = useState('');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!activeTeamId) return;
    setLoading(true);
    const params: Record<string, string> = {};
    if (status) params.status = status;
    api.listWorkItems(params).then(setItems).catch(() => {}).finally(() => setLoading(false));
  }, [activeTeamId, status]);

  return (
    <div className="space-y-4">
      <div className="flex justify-between items-center">
        <h1 className="text-2xl font-bold">Queue</h1>
        <div className="flex gap-2">
          <select value={status} onChange={e => setStatus(e.target.value)} className="text-sm">
            <option value="">All Statuses</option>
            <option value="open">Open</option>
            <option value="in_progress">In Progress</option>
            <option value="blocked">Blocked</option>
            <option value="resolved">Resolved</option>
            <option value="closed">Closed</option>
          </select>
          {hasPermission('work.items.create') && (
            <button onClick={() => nav('/work-items/new')} className="text-sm">+ New</button>
          )}
        </div>
      </div>
      {loading ? <p className="text-[var(--text-muted)]">Loading...</p> : (
        <div className="card">
          <table className="w-full text-sm">
            <thead><tr className="text-left text-[var(--text-muted)] border-b border-[var(--border)]">
              <th className="pb-2">Title</th><th className="pb-2">Type</th><th className="pb-2">Status</th><th className="pb-2">Priority</th>
            </tr></thead>
            <tbody>
              {items.map(wi => (
                <tr key={wi.id} className="border-b border-[var(--border)] hover:bg-[var(--border)] cursor-pointer" onClick={() => nav(`/objects/${wi.id}`)}>
                  <td className="py-2">{wi.title}</td>
                  <td className="py-2"><span className="badge badge-gray">{wi.work_item_type}</span></td>
                  <td className="py-2"><span className={`badge ${wi.status === 'open' ? 'badge-blue' : wi.status === 'blocked' ? 'badge-red' : wi.status === 'resolved' ? 'badge-green' : 'badge-gray'}`}>{wi.status}</span></td>
                  <td className="py-2">{wi.priority}</td>
                </tr>
              ))}
              {items.length === 0 && <tr><td colSpan={4} className="py-4 text-center text-[var(--text-muted)]">No items found</td></tr>}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
