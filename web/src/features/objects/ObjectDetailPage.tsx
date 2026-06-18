import { useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Trash2, ArrowLeft } from 'lucide-react';
import { api } from '@/api/client';
import { keys } from '@/api/keys';
import { useAuth } from '@/auth/context';
import { Card } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { StatusBadge } from '@/components/ui/status-badge';
import { notify } from '@/components/Toaster';
import { InlineSpinner, ErrorState, EmptyState } from '@/components/PageState';
import ObjectEditForm from './ObjectEditForm';

function statusTone(status: string): 'success' | 'danger' | 'info' | 'warning' | 'neutral' {
  if (status === 'resolved' || status === 'closed') return 'success';
  if (status === 'blocked') return 'danger';
  if (status === 'in_progress') return 'info';
  if (status === 'open') return 'warning';
  return 'neutral';
}

export default function ObjectDetailPage() {
  const { id } = useParams<{ id: string }>();
  const nav = useNavigate();
  const { hasPermission, activeTeamId } = useAuth();
  const queryClient = useQueryClient();
  const [newComment, setNewComment] = useState('');
  const [editing, setEditing] = useState(false);

  // Thread the real teamId through the query keys so WS invalidation
  // (['teams', teamId, ...]) reaches these queries on object/comment changes.
  const teamId = activeTeamId ?? '';
  const objQ = useQuery({
    queryKey: keys.objects.detail(teamId, id ?? ''),
    queryFn: ({ signal }) => api.getObject(id!, signal),
    enabled: !!id,
  });
  const commentsQ = useQuery({
    queryKey: keys.objects.comments(teamId, id ?? ''),
    queryFn: () => api.listComments(id!),
    enabled: !!id,
  });
  const linksQ = useQuery({
    queryKey: keys.objects.links(teamId, id ?? ''),
    queryFn: () => api.listLinks(id!),
    enabled: !!id,
  });

  const addCommentMut = useMutation({
    mutationFn: (body: string) => api.createComment(id!, body),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: keys.objects.comments(teamId, id ?? '') });
      setNewComment('');
    },
    onError: (err) => notify.mutationError('Comment', err),
  });

  const deleteMut = useMutation({
    mutationFn: () => api.deleteObject(id!),
    onSuccess: () => {
      notify.success('Deleted');
      nav('/queue');
    },
    onError: (err) => notify.mutationError('Delete', err),
  });

  const obj = objQ.data;
  const refreshObject = () => {
    queryClient.invalidateQueries({ queryKey: keys.objects.detail(teamId, id ?? '') });
    setEditing(false);
  };

  if (objQ.isPending) return <InlineSpinner />;
  if (objQ.error) return <ErrorState message="Failed to load object" onRetry={() => objQ.refetch()} />;
  if (!obj) return <EmptyState title="Object not found" />;

  const comments = commentsQ.data ?? [];
  const links = linksQ.data ?? [];

  const addComment = () => {
    if (!newComment.trim()) return;
    addCommentMut.mutate(newComment.trim());
  };

  return (
    <div className="space-y-4">
      <button
        type="button"
        onClick={() => nav(-1)}
        className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
      >
        <ArrowLeft className="size-4" /> Back
      </button>

      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="font-heading text-2xl font-semibold tracking-tight">{obj.title}</h1>
          <div className="mt-1 flex items-center gap-2">
            <StatusBadge tone="neutral">{obj.object_type}</StatusBadge>
            <StatusBadge tone={statusTone(obj.status)}>{obj.status.replace('_', ' ')}</StatusBadge>
            <span className="text-xs text-muted-foreground">v{obj.version}</span>
          </div>
        </div>
        <div className="flex gap-2">
          {hasPermission('objects.update') && !editing && (
            <Button variant="secondary" size="sm" onClick={() => setEditing(true)}>Edit</Button>
          )}
          {hasPermission('objects.delete') && (
            <Button
              variant="destructive" size="sm"
              data-testid="obj-delete"
              onClick={() => { if (confirm('Delete this object?')) deleteMut.mutate(); }}
            >
              <Trash2 className="size-4" /> Delete
            </Button>
          )}
        </div>
      </div>

      <div className="grid gap-4 lg:grid-cols-3">
        <div className="space-y-4 lg:col-span-2">
          {editing ? (
            <ObjectEditForm obj={obj} onUpdated={refreshObject} onCancel={() => setEditing(false)} />
          ) : (
            <Card className="p-5">
              <h3 className="mb-2 font-heading text-sm font-semibold">Details</h3>
              <dl className="grid grid-cols-3 gap-2 text-sm">
                <dt className="text-muted-foreground">Summary</dt>
                <dd className="col-span-2">{obj.summary || '—'}</dd>
                <dt className="text-muted-foreground">Priority</dt>
                <dd className="col-span-2 capitalize">{obj.priority}</dd>
                <dt className="text-muted-foreground">Created</dt>
                <dd className="col-span-2">{new Date(obj.created_at).toLocaleString()}</dd>
              </dl>
            </Card>
          )}

          <Card className="p-5">
            <h3 className="mb-3 font-heading text-sm font-semibold">Comments ({comments.length})</h3>
            {hasPermission('objects.comments.create') && (
              <div className="mb-4 flex gap-2">
                <Input
                  placeholder="Add a comment…"
                  value={newComment}
                  data-testid="comment-input"
                  onChange={e => setNewComment(e.target.value)}
                  onKeyDown={e => { if (e.key === 'Enter') addComment(); }}
                />
                <Button size="sm" data-testid="comment-post" disabled={!newComment.trim() || addCommentMut.isPending} onClick={addComment}>
                  Post
                </Button>
              </div>
            )}
            {comments.length === 0 ? (
              <p className="py-2 text-sm text-muted-foreground">No comments yet.</p>
            ) : (
              <div className="divide-y divide-border">
                {comments.map(c => (
                  <div key={c.id} className="py-2">
                    <div className="text-xs text-muted-foreground">
                      {c.author_id?.slice(0, 8)}… · {new Date(c.created_at).toLocaleString()}
                    </div>
                    <div className="mt-1 text-sm">{c.body}</div>
                  </div>
                ))}
              </div>
            )}
          </Card>
        </div>

        <Card className="h-fit p-5">
          <h3 className="mb-2 font-heading text-sm font-semibold">Links ({links.length})</h3>
          {links.length === 0 ? (
            <p className="text-sm text-muted-foreground">No linked objects.</p>
          ) : (
            links.map(l => {
              const targetId = l.to_object_id === id ? l.from_object_id : l.to_object_id;
              return (
                <button
                  key={l.id}
                  type="button"
                  onClick={() => nav(`/objects/${targetId}`)}
                  className="block w-full py-1 text-left text-sm text-primary hover:underline"
                >
                  {l.relation_type} → {targetId.slice(0, 8)}…
                </button>
              );
            })
          )}
        </Card>
      </div>
    </div>
  );
}
