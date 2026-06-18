import { useQuery } from '@tanstack/react-query';
import { api } from '@/api/client';
import { keys } from '@/api/keys';
import { Card } from '@/components/ui/card';
import {
  Table, TableHeader, TableBody, TableHead, TableRow, TableCell,
} from '@/components/ui/table';
import { TableSkeleton, ErrorState, EmptyState } from '@/components/PageState';

export default function AdminAudit() {
  const { data: events, isPending, error, refetch } = useQuery({
    queryKey: keys.admin.audit(),
    queryFn: () => api.listAudit(),
  });

  const list = events ?? [];

  return (
    <div className="space-y-4">
      <h1 className="font-heading text-2xl font-semibold tracking-tight">Audit Log</h1>
      <Card className="p-0">
        {isPending ? (
          <div className="p-4"><TableSkeleton rows={8} cols={4} /></div>
        ) : error ? (
          <div className="p-4"><ErrorState message="Failed to load audit log" onRetry={() => refetch()} /></div>
        ) : list.length === 0 ? (
          <div className="p-4"><EmptyState title="No audit events" /></div>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Time</TableHead><TableHead>Actor</TableHead><TableHead>Action</TableHead><TableHead>Entity</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {list.map(e => (
                <TableRow key={e.id || e.created_at}>
                  <TableCell className="text-muted-foreground">{new Date(e.created_at).toLocaleString()}</TableCell>
                  <TableCell>{e.actor_id?.slice(0, 8) || 'system'}…</TableCell>
                  <TableCell className="font-medium">{e.action}</TableCell>
                  <TableCell className="text-muted-foreground">{e.entity_type}/{e.entity_id?.slice(0, 8)}…</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </Card>
    </div>
  );
}
