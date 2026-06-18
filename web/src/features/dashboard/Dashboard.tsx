import { useNavigate } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import {
  ListChecks, Flame, Activity, Plus, AlertTriangle, Kanban,
} from 'lucide-react';
import { api } from '@/api/client';
import { keys } from '@/api/keys';
import { useAuth } from '@/auth/context';
import { Card } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { StatusBadge } from '@/components/ui/status-badge';
import { TableSkeleton, EmptyState } from '@/components/PageState';

function statusTone(status: string): 'success' | 'danger' | 'info' | 'warning' | 'neutral' {
  if (status === 'resolved' || status === 'closed') return 'success';
  if (status === 'blocked') return 'danger';
  if (status === 'in_progress') return 'info';
  if (status === 'open') return 'warning';
  return 'neutral';
}

export default function Dashboard() {
  const { user, activeTeamId, hasPermission } = useAuth();
  const nav = useNavigate();

  // Two independent queries — React Query isolates errors per-key, replacing
  // the old Promise.all with per-call .catch(()=>[]) that silently swallowed failures.
  const workItemsQuery = useQuery({
    queryKey: keys.workItems.list(activeTeamId ?? '', { status: 'open' }),
    queryFn: ({ signal }) => api.listWorkItems({ status: 'open' }, signal),
    enabled: !!activeTeamId,
  });
  const incidentsQuery = useQuery({
    queryKey: keys.incidents.list(activeTeamId ?? ''),
    queryFn: ({ signal }) => api.listIncidents(signal),
    enabled: !!activeTeamId,
  });

  const workItems = workItemsQuery.data ?? [];
  const openIncidents = (incidentsQuery.data ?? []).filter(
    i => i.status !== 'resolved' && i.status !== 'closed',
  );

  const statsLoading = workItemsQuery.isPending || incidentsQuery.isPending;
  const workItemsError = workItemsQuery.error;
  const incidentsError = incidentsQuery.error;

  const quickActions = [
    hasPermission('work.items.create') && {
      label: 'New Work Item', icon: Plus, tone: 'primary' as const, path: '/work-items/new',
    },
    hasPermission('incidents.create') && {
      label: 'Report Incident', icon: Flame, tone: 'danger' as const, path: '/incidents',
    },
    {
      label: 'View Queue', icon: ListChecks, tone: 'muted' as const, path: '/queue',
    },
    {
      label: 'Open Board', icon: Kanban, tone: 'muted' as const, path: '/board',
    },
  ].filter(Boolean) as { label: string; icon: typeof Plus; tone: 'primary' | 'danger' | 'muted'; path: string }[];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="font-heading text-2xl font-semibold tracking-tight">Dashboard</h1>
        <p className="text-sm text-muted-foreground">Welcome back, {user?.name || 'User'}</p>
      </div>

      {/* Stat cards */}
      {statsLoading ? (
        <div className="grid gap-4 sm:grid-cols-3">
          {[0, 1, 2].map(i => <Card key={i} className="p-5"><TableSkeleton rows={2} cols={1} /></Card>)}
        </div>
      ) : (
        <div className="grid gap-4 sm:grid-cols-3">
          <Card className="p-5">
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium text-muted-foreground">Open Work Items</span>
              <ListChecks className="size-4 text-muted-foreground" />
            </div>
            <div className="mt-2 text-3xl font-bold text-primary">
              {workItemsError ? '—' : workItems.length}
            </div>
            {workItemsError && (
              <p className="mt-1 text-xs text-destructive">Failed to load</p>
            )}
          </Card>
          <Card className="p-5">
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium text-muted-foreground">Active Incidents</span>
              <Flame className="size-4 text-muted-foreground" />
            </div>
            <div className="mt-2 text-3xl font-bold text-destructive">
              {incidentsError ? '—' : openIncidents.length}
            </div>
            {incidentsError && (
              <p className="mt-1 text-xs text-destructive">Failed to load</p>
            )}
          </Card>
          <Card className="p-5">
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium text-muted-foreground">System</span>
              <Activity className="size-4 text-muted-foreground" />
            </div>
            <div className="mt-2 flex items-center gap-2">
              <StatusBadge tone="success" dot>Operational</StatusBadge>
            </div>
          </Card>
        </div>
      )}

      <div className="grid gap-4 lg:grid-cols-2">
        {/* Recent work items */}
        <Card className="p-5">
          <div className="mb-3 flex items-center justify-between">
            <h3 className="font-heading text-sm font-semibold">Recent Work Items</h3>
            <Button variant="ghost" size="sm" onClick={() => nav('/queue')}>View all</Button>
          </div>
          {workItemsQuery.isPending ? (
            <TableSkeleton rows={4} cols={1} />
          ) : workItemsError ? (
            <p className="py-4 text-center text-sm text-destructive">
              <AlertTriangle className="mx-auto mb-1 size-4" /> Failed to load work items
            </p>
          ) : workItems.length === 0 ? (
            <EmptyState title="No open work items" />
          ) : (
            <div className="divide-y divide-border">
              {workItems.slice(0, 5).map(wi => (
                <button
                  key={wi.id}
                  type="button"
                  onClick={() => nav(`/objects/${wi.id}`)}
                  className="flex w-full items-center justify-between py-2 text-left hover:bg-accent/50 -mx-2 px-2 rounded-sm"
                >
                  <span className="truncate text-sm">{wi.title}</span>
                  <StatusBadge tone={statusTone(wi.status)}>{wi.status.replace('_', ' ')}</StatusBadge>
                </button>
              ))}
            </div>
          )}
        </Card>

        {/* Quick actions */}
        <Card className="p-5">
          <h3 className="mb-3 font-heading text-sm font-semibold">Quick Actions</h3>
          <div className="grid gap-2">
            {quickActions.map(action => {
              const Icon = action.icon;
              return (
                <button
                  key={action.label}
                  type="button"
                  onClick={() => nav(action.path)}
                  className={`flex items-center gap-2 rounded-md border border-border px-3 py-2 text-sm font-medium transition-colors hover:bg-accent ${
                    action.tone === 'primary' ? 'text-primary' :
                    action.tone === 'danger' ? 'text-destructive' :
                    'text-muted-foreground'
                  }`}
                >
                  <Icon className="size-4" /> {action.label}
                </button>
              );
            })}
          </div>
        </Card>
      </div>
    </div>
  );
}
