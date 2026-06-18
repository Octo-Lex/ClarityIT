import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '@/api/client';
import { keys } from '@/api/keys';
import { Card } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { StatusBadge } from '@/components/ui/status-badge';
import {
  Table, TableHeader, TableBody, TableHead, TableRow, TableCell,
} from '@/components/ui/table';
import { notify } from '@/components/Toaster';
import { TableSkeleton, ErrorState, EmptyState } from '@/components/PageState';

export default function AdminUsers() {
  const queryClient = useQueryClient();
  const { data: users, isPending, error, refetch } = useQuery({
    queryKey: keys.admin.users(),
    queryFn: () => api.listUsers(),
  });

  const toggleMut = useMutation({
    mutationFn: ({ id, active }: { id: string; active: boolean }) =>
      api.updateUser(id, { is_active: !active }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: keys.admin.users() });
      notify.success('User updated');
    },
    onError: (err) => notify.mutationError('Update user', err),
  });

  const list = users ?? [];

  return (
    <div className="space-y-4">
      <h1 className="font-heading text-2xl font-semibold tracking-tight">Users</h1>
      <Card className="p-0">
        {isPending ? (
          <div className="p-4"><TableSkeleton rows={5} cols={5} /></div>
        ) : error ? (
          <div className="p-4"><ErrorState message="Failed to load users" onRetry={() => refetch()} /></div>
        ) : list.length === 0 ? (
          <div className="p-4"><EmptyState title="No users" /></div>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead><TableHead>Email</TableHead><TableHead>Role</TableHead><TableHead>Status</TableHead><TableHead>Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {list.map(u => (
                <TableRow key={u.id}>
                  <TableCell className="font-medium">{u.name}</TableCell>
                  <TableCell className="text-muted-foreground">{u.email}</TableCell>
                  <TableCell>
                    <StatusBadge tone={(u as { is_platform_owner?: boolean }).is_platform_owner ? 'warning' : 'neutral'}>
                      {(u as { is_platform_owner?: boolean }).is_platform_owner ? 'Owner' : 'User'}
                    </StatusBadge>
                  </TableCell>
                  <TableCell>
                    <StatusBadge tone={u.active ? 'success' : 'danger'}>{u.active ? 'Active' : 'Inactive'}</StatusBadge>
                  </TableCell>
                  <TableCell>
                    <Button
                      size="sm" variant="secondary"
                      data-testid={`user-toggle-${u.id}`}
                      onClick={() => toggleMut.mutate({ id: u.id, active: u.active })}
                    >
                      {u.active ? 'Deactivate' : 'Activate'}
                    </Button>
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
