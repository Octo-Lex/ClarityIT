import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { api } from '../../api/client';
import { useAuth } from '../../auth/context';

export default function WorkItemNew() {
  const nav = useNavigate();
  const { activeTeamId } = useAuth();
  const [title, setTitle] = useState('');
  const [summary, setSummary] = useState('');
  const [workType, setWorkType] = useState('task');
  const [priority, setPriority] = useState('none');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!activeTeamId) return;
    setLoading(true); setError('');
    try {
      const r = await api.createWorkItem({ title, summary, work_item_type: workType, priority, status: 'open' });
      nav(`/objects/${r.id}`, { replace: true });
    } catch (e: any) { setError(e.message); } finally { setLoading(false); }
  };

  return (
    <div className="max-w-lg mx-auto space-y-4">
      <h1 className="text-2xl font-bold">New Work Item</h1>
      {error && <div className="error-msg">{error}</div>}
      <form onSubmit={submit} className="card space-y-3">
        <input placeholder="Title *" value={title} onChange={e => setTitle(e.target.value)} required />
        <textarea placeholder="Summary" value={summary} onChange={e => setSummary(e.target.value)} rows={3} className="w-full" />
        <div className="grid grid-cols-2 gap-3">
          <select value={workType} onChange={e => setWorkType(e.target.value)}>
            <option value="task">Task</option><option value="bug">Bug</option>
            <option value="ticket">Ticket</option><option value="change">Change</option>
          </select>
          <select value={priority} onChange={e => setPriority(e.target.value)}>
            <option value="none">No Priority</option><option value="low">Low</option>
            <option value="medium">Medium</option><option value="high">High</option><option value="critical">Critical</option>
          </select>
        </div>
        <div className="flex gap-2">
          <button type="submit" disabled={loading}>{loading ? 'Creating...' : 'Create'}</button>
          <button type="button" className="btn-secondary" onClick={() => nav(-1)}>Cancel</button>
        </div>
      </form>
    </div>
  );
}
