import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { api, type ObjectDetail, type Comment, type Link } from '../../api/client';
import { useAuth } from '../../auth/context';
import ObjectEditForm from './ObjectEditForm';

export default function ObjectDetailPage() {
  const { id } = useParams<{ id: string }>();
  const nav = useNavigate();
  const { hasPermission } = useAuth();
  const [obj, setObj] = useState<ObjectDetail | null>(null);
  const [comments, setComments] = useState<Comment[]>([]);
  const [links, setLinks] = useState<Link[]>([]);
  const [newComment, setNewComment] = useState('');
  const [editing, setEditing] = useState(false);
  const [error, setError] = useState('');

  const load = () => {
    if (!id) return;
    Promise.all([api.getObject(id), api.listComments(id), api.listLinks(id)])
      .then(([o, c, l]) => { setObj(o); setComments(c); setLinks(l); })
      .catch(e => setError(e.message));
  };
  useEffect(load, [id]);

  const addComment = async () => {
    if (!id || !newComment.trim()) return;
    try { await api.createComment(id, newComment); setNewComment(''); load(); } catch (e: any) { setError(e.message); }
  };

  const deleteObj = async () => {
    if (!id || !confirm('Delete this object?')) return;
    try { await api.deleteObject(id); nav('/queue'); } catch (e: any) { setError(e.message); }
  };

  if (!obj) return <p className="text-[var(--text-muted)]">{error || 'Loading...'}</p>;

  return (
    <div className="space-y-4">
      {error && <div className="error-msg">{error}</div>}
      <div className="flex justify-between items-start">
        <div>
          <h1 className="text-2xl font-bold">{obj.title}</h1>
          <div className="flex gap-2 mt-1">
            <span className="badge badge-gray">{obj.object_type}</span>
            <span className="badge badge-blue">{obj.status}</span>
            <span className="text-sm text-[var(--text-muted)]">v{obj.version}</span>
          </div>
        </div>
        {hasPermission('objects.update') && !editing && <button className="btn-secondary text-sm" onClick={() => setEditing(true)}>Edit</button>}
        {hasPermission('objects.delete') && <button className="btn-danger text-sm" onClick={deleteObj}>Delete</button>}
      </div>

      <div className="grid grid-cols-3 gap-4">
        <div className="col-span-2 space-y-4">
          {editing ? (
            <ObjectEditForm obj={obj} onUpdated={() => { setEditing(false); load(); }} onCancel={() => setEditing(false)} />
          ) : (
          <div className="card">
            <h3 className="font-semibold mb-2">Details</h3>
            <dl className="grid grid-cols-2 gap-2 text-sm">
              <dt className="text-[var(--text-muted)]">Summary</dt><dd>{obj.summary || '—'}</dd>
              <dt className="text-[var(--text-muted)]">Priority</dt><dd>{obj.priority}</dd>
              <dt className="text-[var(--text-muted)]">Created</dt><dd>{new Date(obj.created_at).toLocaleString()}</dd>
            </dl>
          </div>
          )}

          <div className="card">
            <h3 className="font-semibold mb-3">Comments ({comments.length})</h3>
            {hasPermission('objects.comments.create') && (
              <div className="flex gap-2 mb-4">
                <input placeholder="Add a comment..." value={newComment} onChange={e => setNewComment(e.target.value)} onKeyDown={e => e.key === 'Enter' && addComment()} />
                <button onClick={addComment} disabled={!newComment.trim()}>Post</button>
              </div>
            )}
            {comments.map(c => (
              <div key={c.id} className="border-b border-[var(--border)] py-2 last:border-0">
                <div className="text-xs text-[var(--text-muted)]">{c.author_id?.slice(0,8)}… · {new Date(c.created_at).toLocaleString()}</div>
                <div className="text-sm mt-1">{c.body}</div>
              </div>
            ))}
          </div>
        </div>

        <div className="space-y-4">
          <div className="card">
            <h3 className="font-semibold mb-2">Links ({links.length})</h3>
            {links.map(l => {
              const targetId = l.to_object_id === id ? l.from_object_id : l.to_object_id;
              return <a key={l.id} href={`/objects/${targetId}`} onClick={e => { e.preventDefault(); nav(`/objects/${targetId}`); }} className="block text-sm text-[var(--primary)] hover:underline py-1">
                {l.relation_type} → {targetId.slice(0,8)}…
              </a>;
            })}
          </div>
        </div>
      </div>
    </div>
  );
}
