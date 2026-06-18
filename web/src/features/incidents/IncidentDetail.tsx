import { useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { ArrowLeft, CheckCircle2 } from 'lucide-react';
import { api } from '@/api/client';
import { keys } from '@/api/keys';
import { Perm } from '@/auth/permissions';
import { useAuth } from '@/auth/context';
import { Card } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { StatusBadge } from '@/components/ui/status-badge';
import { notify } from '@/components/Toaster';
import { InlineSpinner, ErrorState, EmptyState } from '@/components/PageState';
import { RelatedKnowledgePanel } from '../knowledge/RelatedKnowledgePanel';

function sevTone(sev: string): 'danger' | 'warning' | 'info' | 'neutral' {
  if (sev === 'sev1') return 'danger';
  if (sev === 'sev2') return 'warning';
  if (sev === 'sev3') return 'info';
  return 'neutral';
}
function statusTone(status: string): 'success' | 'info' | 'neutral' {
  if (status === 'resolved' || status === 'closed') return 'success';
  if (status === 'open') return 'info';
  return 'neutral';
}

export default function IncidentDetail() {
  const { id } = useParams<{ id: string }>();
  const nav = useNavigate();
  const { hasPermission, activeTeamId } = useAuth();
  const queryClient = useQueryClient();
  const [timeline, setTimeline] = useState('');

  const teamId = activeTeamId ?? '';
  const incQ = useQuery({
    queryKey: keys.incidents.detail(teamId, id ?? ''),
    queryFn: ({ signal }) => api.getIncident(id!, signal),
    enabled: !!id,
  });
  const commentsQ = useQuery({
    queryKey: keys.objects.comments(teamId, id ?? ''),
    queryFn: () => api.listComments(id!),
    enabled: !!id,
  });

  // Resolve uses expected_version for optimistic concurrency (preserved from
  // the original). A 409 means someone else changed it; we invalidate + notify.
  const resolveMut = useMutation({
    mutationFn: () => api.updateIncident(id!, { status: 'resolved', expected_version: incQ.data!.version }),
    onSuccess: () => {
      notify.success('Incident resolved');
      queryClient.invalidateQueries({ queryKey: keys.incidents.detail(teamId, id ?? '') });
      queryClient.invalidateQueries({ queryKey: keys.incidents.list(teamId) });
    },
    onError: (err) => {
      // 409 → refresh to latest; other errors → toast.
      if (err instanceof Error && 'status' in err && err.status === 409) {
        notify.warning('Version conflict', 'Refreshing to the latest version…');
        queryClient.invalidateQueries({ queryKey: keys.incidents.detail(teamId, id ?? '') });
      } else {
        notify.mutationError('Resolve', err);
      }
    },
  });

  // Timeline entry = a comment on the incident (original behavior).
  const addTimelineMut = useMutation({
    mutationFn: (body: string) => api.addTimeline(id!, body),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: keys.objects.comments(teamId, id ?? '') });
      setTimeline('');
    },
    onError: (err) => notify.mutationError('Add timeline entry', err),
  });

  const inc = incQ.data;
  const comments = commentsQ.data ?? [];
  const canUpdate = hasPermission(Perm.IncidentsUpdate);

  if (incQ.isPending) return <InlineSpinner />;
  if (incQ.error) return <ErrorState message="Failed to load incident" onRetry={() => incQ.refetch()} />;
  if (!inc) return <EmptyState title="Incident not found" />;

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
          <h1 className="font-heading text-2xl font-semibold tracking-tight">{inc.title}</h1>
          <div className="mt-1 flex items-center gap-2">
            <StatusBadge tone={sevTone(inc.severity)}>{inc.severity.toUpperCase()}</StatusBadge>
            <StatusBadge tone={statusTone(inc.status)}>{inc.status.replace('_', ' ')}</StatusBadge>
            <span className="text-xs text-muted-foreground">v{inc.version}</span>
          </div>
        </div>
        {canUpdate && inc.status !== 'resolved' && (
          <Button size="sm" data-testid="inc-resolve" disabled={resolveMut.isPending} onClick={() => resolveMut.mutate()}>
            <CheckCircle2 className="size-4" /> {resolveMut.isPending ? 'Resolving…' : 'Resolve'}
          </Button>
        )}
      </div>

      <div className="grid gap-4 lg:grid-cols-3">
        <div className="space-y-4 lg:col-span-2">
          <Card className="p-5">
            <h3 className="mb-2 font-heading text-sm font-semibold">Details</h3>
            <dl className="grid grid-cols-3 gap-2 text-sm">
              <dt className="text-muted-foreground">Summary</dt>
              <dd className="col-span-2">{inc.summary || '—'}</dd>
              <dt className="text-muted-foreground">Impact</dt>
              <dd className="col-span-2">{inc.impact || '—'}</dd>
              <dt className="text-muted-foreground">Created</dt>
              <dd className="col-span-2">{new Date(inc.created_at).toLocaleString()}</dd>
              {inc.resolved_at && (
                <>
                  <dt className="text-muted-foreground">Resolved</dt>
                  <dd className="col-span-2">{new Date(inc.resolved_at).toLocaleString()}</dd>
                </>
              )}
            </dl>
          </Card>

          <Card className="p-5">
            <h3 className="mb-3 font-heading text-sm font-semibold">Timeline</h3>
            {canUpdate && (
              <div className="mb-4 flex gap-2">
                <Input
                  placeholder="Add timeline entry…"
                  value={timeline}
                  data-testid="timeline-input"
                  onChange={e => setTimeline(e.target.value)}
                  onKeyDown={e => { if (e.key === 'Enter' && timeline.trim()) addTimelineMut.mutate(timeline.trim()); }}
                />
                <Button
                  size="sm" data-testid="timeline-add"
                  disabled={!timeline.trim() || addTimelineMut.isPending}
                  onClick={() => timeline.trim() && addTimelineMut.mutate(timeline.trim())}
                >
                  Add
                </Button>
              </div>
            )}
            {comments.length === 0 ? (
              <p className="py-2 text-sm text-muted-foreground">No timeline entries yet.</p>
            ) : (
              <div className="space-y-2">
                {comments.map(c => (
                  <div key={c.id} className="border-l-2 border-primary pl-3 py-1.5">
                    <div className="text-xs text-muted-foreground">{new Date(c.created_at).toLocaleString()}</div>
                    <div className="mt-0.5 text-sm">{c.body}</div>
                  </div>
                ))}
              </div>
            )}
          </Card>
        </div>

        {/* v1.5 Track 4: Related Knowledge Panel */}
        <RelatedKnowledgePanel sourceType="incident" sourceId={id || ''} />
      </div>
    </div>
  );
}
