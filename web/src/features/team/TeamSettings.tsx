import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { UserPlus, Trash2 } from 'lucide-react';
import { api } from '@/api/client';
import { keys } from '@/api/keys';
import { Perm } from '@/auth/permissions';
import { useAuth } from '@/auth/context';
import { Card } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { StatusBadge } from '@/components/ui/status-badge';
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '@/components/ui/select';
import {
  Table, TableHeader, TableBody, TableHead, TableRow, TableCell,
} from '@/components/ui/table';
import { notify } from '@/components/Toaster';
import { TableSkeleton, ErrorState } from '@/components/PageState';

function roleTone(role: string): 'warning' | 'danger' | 'info' | 'neutral' {
  if (role === 'owner') return 'warning';
  if (role === 'admin') return 'danger';
  if (role === 'manager') return 'info';
  return 'neutral';
}

const INVITE_ROLES = [
  { value: 'member', label: 'Member' },
  { value: 'viewer', label: 'Viewer' },
  { value: 'manager', label: 'Manager' },
  { value: 'admin', label: 'Admin' },
];

export default function TeamSettings() {
  const { activeTeamId, hasPermission } = useAuth();
  const queryClient = useQueryClient();
  const [inviteEmail, setInviteEmail] = useState('');
  const [inviteRole, setInviteRole] = useState('member');

  const teamId = activeTeamId ?? '';
  const membersQ = useQuery({
    queryKey: keys.members(teamId),
    queryFn: () => api.listMembers(),
    enabled: !!activeTeamId,
  });
  const invitationsQ = useQuery({
    queryKey: keys.invitations(teamId),
    queryFn: () => api.listInvitations(),
    enabled: !!activeTeamId,
  });

  const inviteMut = useMutation({
    mutationFn: () => api.createInvitation(inviteEmail, inviteRole),
    onSuccess: () => {
      notify.success('Invitation sent');
      queryClient.invalidateQueries({ queryKey: keys.invitations(teamId) });
      setInviteEmail('');
    },
    onError: (err) => notify.mutationError('Invite', err),
  });

  const removeMut = useMutation({
    mutationFn: (userId: string) => api.removeMember(userId),
    onSuccess: () => {
      notify.success('Member removed');
      queryClient.invalidateQueries({ queryKey: keys.members(teamId) });
    },
    onError: (err) => notify.mutationError('Remove member', err),
  });

  const members = membersQ.data ?? [];
  const invitations = (invitationsQ.data ?? []).filter(i => !i.accepted_at);
  const loading = membersQ.isPending || invitationsQ.isPending;

  if (loading) {
    return (
      <div className="space-y-6">
        <h1 className="font-heading text-2xl font-semibold tracking-tight">Team Settings</h1>
        <Card className="p-4"><TableSkeleton rows={4} cols={5} /></Card>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <h1 className="font-heading text-2xl font-semibold tracking-tight">Team Settings</h1>

      <Card className="p-5">
        <div className="mb-3 flex items-center justify-between">
          <h3 className="font-heading text-sm font-semibold">Members ({members.length})</h3>
        </div>
        {membersQ.error ? (
          <ErrorState message="Failed to load members" onRetry={() => membersQ.refetch()} />
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Email</TableHead>
                <TableHead>Role</TableHead>
                <TableHead>Joined</TableHead>
                <TableHead className="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {members.map(m => (
                <TableRow key={m.user_id}>
                  <TableCell className="font-medium">{m.name}</TableCell>
                  <TableCell className="text-muted-foreground">{m.email}</TableCell>
                  <TableCell><StatusBadge tone={roleTone(m.role)}>{m.role}</StatusBadge></TableCell>
                  <TableCell className="text-muted-foreground">{new Date(m.joined_at).toLocaleDateString()}</TableCell>
                  <TableCell className="text-right">
                    {hasPermission(Perm.TeamMembersRemove) && m.role !== 'owner' && (
                      <Button
                        size="sm" variant="destructive"
                        onClick={() => { if (confirm('Remove this member?')) removeMut.mutate(m.user_id); }}
                      >
                        <Trash2 className="size-4" /> Remove
                      </Button>
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </Card>

      {hasPermission(Perm.TeamInvitationsCreate) && (
        <Card className="p-5">
          <h3 className="mb-3 font-heading text-sm font-semibold">Invite Member</h3>
          <form
            className="flex flex-wrap items-end gap-3"
            onSubmit={(e) => { e.preventDefault(); if (inviteEmail.trim()) inviteMut.mutate(); }}
          >
            <div className="min-w-[200px] flex-1 space-y-1.5">
              <Label htmlFor="invite-email">Email address</Label>
              <Input
                id="invite-email" type="email" data-testid="invite-email"
                value={inviteEmail} onChange={e => setInviteEmail(e.target.value)} required
              />
            </div>
            <div className="w-36 space-y-1.5">
              <Label>Role</Label>
              <Select value={inviteRole} onValueChange={(v) => setInviteRole(v ?? 'member')}>
                <SelectTrigger data-testid="invite-role"><SelectValue /></SelectTrigger>
                <SelectContent>
                  {INVITE_ROLES.map(r => <SelectItem key={r.value} value={r.value}>{r.label}</SelectItem>)}
                </SelectContent>
              </Select>
            </div>
            <Button type="submit" data-testid="invite-submit" disabled={inviteMut.isPending || !inviteEmail.trim()}>
              <UserPlus className="size-4" /> {inviteMut.isPending ? 'Sending…' : 'Invite'}
            </Button>
          </form>
        </Card>
      )}

      {invitations.length > 0 && (
        <Card className="p-5">
          <h3 className="mb-3 font-heading text-sm font-semibold">Pending Invitations</h3>
          <div className="divide-y divide-border">
            {invitations.map(inv => (
              <div key={inv.id} className="flex items-center justify-between py-2">
                <span className="text-sm">{inv.email}</span>
                <div className="flex items-center gap-2">
                  <StatusBadge tone="neutral">{inv.role}</StatusBadge>
                  <span className="text-xs text-muted-foreground">Expires {new Date(inv.expires_at).toLocaleDateString()}</span>
                </div>
              </div>
            ))}
          </div>
        </Card>
      )}
    </div>
  );
}
