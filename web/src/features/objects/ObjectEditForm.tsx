import { useState } from 'react';
import { api, type ObjectDetail, ApiError } from '../../api/client';
import { useAuth } from '../../auth/context';

interface Props {
  obj: ObjectDetail;
  onUpdated: () => void;
  onCancel: () => void;
}

export default function ObjectEditForm({ obj, onUpdated, onCancel }: Props) {
  const { hasPermission } = useAuth();
  const [title, setTitle] = useState(obj.title);
  const [summary, setSummary] = useState(obj.summary || '');
  const [status, setStatus] = useState(obj.status);
  const [priority, setPriority] = useState(obj.priority);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  if (!hasPermission('objects.update')) {
    return <p className="text-sm text-[var(--text-muted)]">No permission to edit.</p>;
  }

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true); setError('');
    try {
      await api.updateObject(obj.id, {
        title, summary, status, priority,
        expected_version: obj.version,
      });
      onUpdated();
    } catch (err: any) {
      if (err?.status === 409) {
        setError('Version conflict — someone else updated this. Refreshing...');
        setTimeout(onUpdated, 1500);
      } else {
        setError(err.message || 'Update failed');
      }
    } finally { setLoading(false); }
  };

  return (
    <form onSubmit={submit} className="card space-y-3">
      {error && <div className="error-msg">{error}</div>}
      <div>
        <label className="text-xs text-[var(--text-muted)]">Title</label>
        <input value={title} onChange={e => setTitle(e.target.value)} required />
      </div>
      <div>
        <label className="text-xs text-[var(--text-muted)]">Summary</label>
        <textarea value={summary} onChange={e => setSummary(e.target.value)} rows={3} className="w-full" />
      </div>
      <div className="grid grid-cols-2 gap-3">
        <div>
          <label className="text-xs text-[var(--text-muted)]">Status</label>
          <select value={status} onChange={e => setStatus(e.target.value)}>
            <option value="open">Open</option><option value="in_progress">In Progress</option>
            <option value="blocked">Blocked</option><option value="resolved">Resolved</option>
            <option value="closed">Closed</option>
          </select>
        </div>
        <div>
          <label className="text-xs text-[var(--text-muted)]">Priority</label>
          <select value={priority} onChange={e => setPriority(e.target.value)}>
            <option value="none">None</option><option value="low">Low</option>
            <option value="medium">Medium</option><option value="high">High</option>
            <option value="critical">Critical</option>
          </select>
        </div>
      </div>
      <div className="flex gap-2">
        <button type="submit" disabled={loading}>{loading ? 'Saving...' : 'Save'}</button>
        <button type="button" className="btn-secondary" onClick={onCancel}>Cancel</button>
      </div>
    </form>
  );
}
