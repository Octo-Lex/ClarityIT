import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { Plus } from 'lucide-react';
import { api } from '@/api/client';
import { keys } from '@/api/keys';
import { useAuth } from '@/auth/context';
import { StatusBadge } from '@/components/ui/status-badge';
import {
  Table, TableHeader, TableBody, TableHead, TableRow, TableCell,
} from '@/components/ui/table';
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Card } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import {
  TableSkeleton, ErrorState, EmptyState,
} from '@/components/PageState';

/** Map a work-item status string to a StatusBadge tone. */
function statusTone(status: string): 'success' | 'danger' | 'info' | 'warning' | 'neutral' {
  if (status === 'resolved' || status === 'closed') return 'success';
  if (status === 'blocked') return 'danger';
  if (status === 'in_progress') return 'info';
  if (status === 'open') return 'warning';
  return 'neutral';
}

function priorityTone(priority: string): 'danger' | 'warning' | 'neutral' {
  if (priority === 'critical' || priority === 'high') return 'danger';
  if (priority === 'medium') return 'warning';
  return 'neutral';
}

const STATUS_TABS = [
  { value: '', label: 'All' },
  { value: 'open', label: 'Open' },
  { value: 'in_progress', label: 'In Progress' },
  { value: 'blocked', label: 'Blocked' },
  { value: 'resolved', label: 'Resolved' },
  { value: 'closed', label: 'Closed' },
];

export default function QueuePage() {
  const { activeTeamId, hasPermission } = useAuth();
  const nav = useNavigate();
  const [status, setStatus] = useState('');

  const { data: items, isPending, error, refetch } = useQuery({
    queryKey: keys.workItems.list(activeTeamId ?? '', status ? { status } : undefined),
    queryFn: ({ signal }) => api.listWorkItems(status ? { status } : undefined, signal),
    enabled: !!activeTeamId,
  });

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1 className="font-heading text-2xl font-semibold tracking-tight">Queue</h1>
        {hasPermission('work.items.create') && (
          <Button size="sm" onClick={() => nav('/work-items/new')}>
            <Plus className="size-4" /> New
          </Button>
        )}
      </div>

      <Tabs value={status} onValueChange={setStatus}>
        <TabsList>
          {STATUS_TABS.map(tab => (
            <TabsTrigger key={tab.value} value={tab.value}>{tab.label}</TabsTrigger>
          ))}
        </TabsList>
      </Tabs>

      <Card className="p-0">
        {isPending ? (
          <div className="p-4"><TableSkeleton rows={5} cols={4} /></div>
        ) : error ? (
          <div className="p-4">
            <ErrorState message="Failed to load queue" onRetry={() => refetch()} />
          </div>
        ) : items && items.length === 0 ? (
          <div className="p-4">
            <EmptyState
              title={status ? `No ${status.replace('_', ' ')} work items` : 'No work items'}
              description="Work items created by your team will appear here."
            />
          </div>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Title</TableHead>
                <TableHead>Type</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Priority</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {items?.map(wi => (
                <TableRow
                  key={wi.id}
                  className="cursor-pointer"
                  onClick={() => nav(`/objects/${wi.id}`)}
                >
                  <TableCell className="font-medium">{wi.title}</TableCell>
                  <TableCell>
                    <StatusBadge tone="neutral">{wi.work_item_type}</StatusBadge>
                  </TableCell>
                  <TableCell>
                    <StatusBadge tone={statusTone(wi.status)}>{wi.status.replace('_', ' ')}</StatusBadge>
                  </TableCell>
                  <TableCell>
                    <StatusBadge tone={priorityTone(wi.priority)}>{wi.priority}</StatusBadge>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </Card>
    </div>
  );
}
