import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { api, type WorkItem, type Incident } from '../../api/client';
import { useAuth } from '../../auth/context';
import { useRefetch } from '../../hooks/useRefetch';

export default function Dashboard() {
  const { user, activeTeamId, hasPermission } = useAuth();
  const nav = useNavigate();
  const { version } = useRefetch();
  const [workItems, setWorkItems] = useState<WorkItem[]>([]);
  const [incidents, setIncidents] = useState<Incident[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!activeTeamId) return;
    setLoading(true);
    Promise.all([api.listWorkItems({ status: 'open' }).catch(() => []), api.listIncidents().catch(() => [])])
      .then(([wi, inc]) => { setWorkItems(wi as WorkItem[]); setIncidents(inc as Incident[]); })
      .finally(() => setLoading(false));
  }, [activeTeamId, version]);

  const openIncidents = incidents.filter(i => i.status !== 'resolved' && i.status !== 'closed');

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Dashboard</h1>
      <p className="text-[var(--text-muted)]">Welcome back, {user?.name || 'User'}</p>

      {loading ? <p className="text-[var(--text-muted)]">Loading...</p> : (
        <div className="grid grid-cols-3 gap-4">
          <div className="card">
            <div className="text-3xl font-bold text-[var(--primary)]">{workItems.length}</div>
            <div className="text-sm text-[var(--text-muted)]">Open Work Items</div>
          </div>
          <div className="card">
            <div className="text-3xl font-bold text-[var(--danger)]">{openIncidents.length}</div>
            <div className="text-sm text-[var(--text-muted)]">Active Incidents</div>
          </div>
          <div className="card">
            <div className="text-3xl font-bold text-[var(--success)]">●</div>
            <div className="text-sm text-[var(--text-muted)]">System Operational</div>
          </div>
        </div>
      )}

      <div className="grid grid-cols-2 gap-4">
        <div className="card">
          <h3 className="font-semibold mb-3">Recent Work Items</h3>
          {workItems.slice(0, 5).map(wi => (
            <div key={wi.id} className="flex justify-between py-2 border-b border-[var(--border)] last:border-0">
              <span className="text-sm">{wi.title}</span>
              <span className="badge badge-blue">{wi.status}</span>
            </div>
          ))}
          {workItems.length === 0 && <p className="text-sm text-[var(--text-muted)]">No open work items</p>}
        </div>
        <div className="card">
          <h3 className="font-semibold mb-3">Quick Actions</h3>
          <div className="space-y-2">
            {hasPermission('work.items.create') && <span onClick={() => nav('/work-items/new')} className="block text-sm text-[var(--primary)] hover:underline cursor-pointer">+ New Work Item</span>}
            {hasPermission('incidents.create') && <span onClick={() => nav('/incidents')} className="block text-sm text-[var(--danger)] hover:underline cursor-pointer">+ Report Incident</span>}
            {hasPermission('objects.create') && <span onClick={() => nav('/queue')} className="block text-sm text-[var(--text-muted)] hover:underline cursor-pointer">View Queue</span>}
            <span onClick={() => nav('/board')} className="block text-sm text-[var(--text-muted)] hover:underline cursor-pointer">Open Board</span>
          </div>
        </div>
      </div>
    </div>
  );
}
