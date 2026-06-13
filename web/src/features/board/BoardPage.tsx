import { useEffect, useState } from 'react';
import { api, type WorkItem } from '../../api/client';
import { useAuth } from '../../auth/context';
import { useNavigate } from 'react-router-dom';

export default function BoardPage() {
  const { activeTeamId } = useAuth();
  const nav = useNavigate();
  const [board, setBoard] = useState<Record<string, WorkItem[]>>({});
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!activeTeamId) return;
    setLoading(true);
    api.getBoard().then(setBoard).catch(() => {}).finally(() => setLoading(false));
  }, [activeTeamId]);

  const statusOrder = ['open', 'in_progress', 'blocked', 'resolved', 'closed'];
  const statusColor: Record<string, string> = {
    open: 'border-t-blue-500', in_progress: 'border-t-yellow-500', blocked: 'border-t-red-500',
    resolved: 'border-t-green-500', closed: 'border-t-gray-500',
  };

  if (loading) return <p className="text-[var(--text-muted)]">Loading board...</p>;

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-bold">Board</h1>
      <div className="flex gap-4 overflow-x-auto pb-4">
        {statusOrder.map(col => (
          <div key={col} className={`min-w-[250px] flex-shrink-0 bg-[var(--card)] rounded-lg border border-[var(--border)] border-t-4 ${statusColor[col] || 'border-t-gray-500'}`}>
            <div className="p-3 font-semibold text-sm border-b border-[var(--border)] flex justify-between">
              <span className="capitalize">{col.replace('_', ' ')}</span>
              <span className="text-[var(--text-muted)]">{(board[col] || []).length}</span>
            </div>
            <div className="p-2 space-y-2 min-h-[200px]">
              {(board[col] || []).map((wi: WorkItem) => (
                <a key={wi.id} href={`/objects/${wi.id}`} onClick={e => { e.preventDefault(); nav(`/objects/${wi.id}`); }} className="block p-3 bg-[var(--bg)] rounded border border-[var(--border)] hover:border-[var(--primary)] transition-colors">
                  <div className="font-medium text-sm">{wi.title}</div>
                  <div className="flex gap-2 mt-1">
                    <span className="badge badge-gray text-xs">{wi.work_item_type}</span>
                    <span className="text-xs text-[var(--text-muted)]">{wi.priority}</span>
                  </div>
                </a>
              ))}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
