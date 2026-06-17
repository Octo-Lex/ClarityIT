import { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { api, type Incident, type Comment } from '../../api/client';
import { useAuth } from '../../auth/context';
import { RelatedKnowledgePanel } from '../knowledge/RelatedKnowledgePanel';

export default function IncidentDetail() {
  const { id } = useParams<{ id: string }>();
  const { hasPermission } = useAuth();
  const [inc, setInc] = useState<Incident | null>(null);
  const [comments, setComments] = useState<Comment[]>([]);
  const [timeline, setTimeline] = useState('');
  const [error, setError] = useState('');

  const load = () => {
    if (!id) return;
    Promise.all([api.getIncident(id), api.listComments(id)]).then(([i, c]) => { setInc(i); setComments(c); }).catch(e => setError(e.message));
  };
  useEffect(load, [id]);

  const resolve = async () => {
    if (!inc || !id) return;
    try {
      await api.updateIncident(id, { status: 'resolved', expected_version: inc.version });
      load();
    } catch (e: any) { setError(e.message); }
  };

  const addTimelineEntry = async () => {
    if (!id || !timeline.trim()) return;
    try { await api.addTimeline(id, timeline); setTimeline(''); load(); } catch (e: any) { setError(e.message); }
  };

  if (!inc) return <p className="text-[var(--text-muted)]">{error || 'Loading...'}</p>;
  const sevColor: Record<string, string> = { sev1: 'badge-red', sev2: 'badge-yellow', sev3: 'badge-blue', sev4: 'badge-gray' };

  return (
    <div className="space-y-4">
      {error && <div className="error-msg">{error}</div>}
      <div className="flex justify-between items-start">
        <div>
          <h1 className="text-2xl font-bold">{inc.title}</h1>
          <div className="flex gap-2 mt-1">
            <span className={`badge ${sevColor[inc.severity]}`}>{inc.severity}</span>
            <span className={`badge ${inc.status === 'resolved' ? 'badge-green' : 'badge-blue'}`}>{inc.status}</span>
            <span className="text-sm text-[var(--text-muted)]">v{inc.version}</span>
          </div>
        </div>
        {hasPermission('incidents.update') && inc.status !== 'resolved' && <button onClick={resolve}>Resolve</button>}
      </div>

      <div className="grid grid-cols-3 gap-4">
        <div className="col-span-2 space-y-4">
          <div className="card">
            <h3 className="font-semibold mb-2">Details</h3>
            <dl className="grid grid-cols-2 gap-2 text-sm">
              <dt className="text-[var(--text-muted)]">Summary</dt><dd>{inc.summary || '—'}</dd>
              <dt className="text-[var(--text-muted)]">Impact</dt><dd>{inc.impact || '—'}</dd>
              <dt className="text-[var(--text-muted)]">Created</dt><dd>{new Date(inc.created_at).toLocaleString()}</dd>
              {inc.resolved_at && <><dt className="text-[var(--text-muted)]">Resolved</dt><dd>{new Date(inc.resolved_at).toLocaleString()}</dd></>}
            </dl>
          </div>

          <div className="card">
            <h3 className="font-semibold mb-3">Timeline</h3>
            {hasPermission('incidents.update') && (
              <div className="flex gap-2 mb-4">
                <input placeholder="Add timeline entry..." value={timeline} onChange={e => setTimeline(e.target.value)} onKeyDown={e => e.key === 'Enter' && addTimelineEntry()} />
                <button onClick={addTimelineEntry} disabled={!timeline.trim()}>Add</button>
              </div>
            )}
            {comments.map(c => (
              <div key={c.id} className="border-l-2 border-[var(--primary)] pl-3 py-2 mb-2">
                <div className="text-xs text-[var(--text-muted)]">{new Date(c.created_at).toLocaleString()}</div>
                <div className="text-sm">{c.body}</div>
              </div>
            ))}
          </div>
        </div>

        {/* v1.5 Track 4: Related Knowledge Panel */}
        <RelatedKnowledgePanel
          sourceType="incident"
          sourceId={id || ''}
        />
      </div>
    </div>
  );
}
