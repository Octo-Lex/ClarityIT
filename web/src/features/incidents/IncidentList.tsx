import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, X } from 'lucide-react';
import { api } from '@/api/client';
import { keys } from '@/api/keys';
import { Perm } from '@/auth/permissions';
import { useAuth } from '@/auth/context';
import { Card } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '@/components/ui/select';
import { StatusBadge } from '@/components/ui/status-badge';
import {
  Table, TableHeader, TableBody, TableHead, TableRow, TableCell,
} from '@/components/ui/table';
import { notify } from '@/components/Toaster';
import { TableSkeleton, ErrorState, EmptyState } from '@/components/PageState';
import PatternCards from './PatternCards';

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

const SEVERITIES = [
  { value: 'sev1', label: 'SEV1 — Critical' },
  { value: 'sev2', label: 'SEV2 — Major' },
  { value: 'sev3', label: 'SEV3 — Minor' },
  { value: 'sev4', label: 'SEV4 — Informational' },
];

export default function IncidentList() {
  const { activeTeamId, hasPermission } = useAuth();
  const nav = useNavigate();
  const queryClient = useQueryClient();
  const [showCreate, setShowCreate] = useState(false);
  const [title, setTitle] = useState('');
  const [severity, setSeverity] = useState('sev3');

  const { data: incidents, isPending, error, refetch } = useQuery({
    queryKey: keys.incidents.list(activeTeamId ?? ''),
    queryFn: ({ signal }) => api.listIncidents(signal),
    enabled: !!activeTeamId,
  });

  // Create via useMutation + precise invalidation (replaces the hand-rolled
  // api.createIncident then api.listIncidents().then(setIncidents) at the old
  // line 29 — no more manual refetch, and errors surface as a toast).
  const createMut = useMutation({
    mutationFn: () => api.createIncident({ title, severity }),
    onSuccess: () => {
      notify.success('Incident created');
      queryClient.invalidateQueries({ queryKey: keys.incidents.list(activeTeamId ?? '') });
      setShowCreate(false);
      setTitle('');
      setSeverity('sev3');
    },
    onError: (err) => notify.mutationError('Create incident', err),
  });

  const canCreate = hasPermission(Perm.IncidentsCreate);
  const list = incidents ?? [];

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="font-heading text-2xl font-semibold tracking-tight">Incidents</h1>
        {canCreate && (
          <Button size="sm" variant={showCreate ? 'secondary' : 'default'} onClick={() => setShowCreate(s => !s)}>
            {showCreate ? <><X className="size-4" /> Cancel</> : <><Plus className="size-4" /> New Incident</>}
          </Button>
        )}
      </div>

      {showCreate && (
        <Card className="space-y-4 p-5">
          <form
            className="space-y-4"
            onSubmit={(e) => { e.preventDefault(); if (title.trim()) createMut.mutate(); }}
          >
            <div className="space-y-1.5">
              <Label htmlFor="inc-title">Title *</Label>
              <Input
                id="inc-title" data-testid="inc-title"
                placeholder="Brief summary of the incident"
                value={title} onChange={e => setTitle(e.target.value)} required
              />
            </div>
            <div className="space-y-1.5">
              <Label>Severity</Label>
              <Select value={severity} onValueChange={(v) => setSeverity(v ?? 'sev3')}>
                <SelectTrigger data-testid="inc-severity"><SelectValue /></SelectTrigger>
                <SelectContent>
                  {SEVERITIES.map(s => <SelectItem key={s.value} value={s.value}>{s.label}</SelectItem>)}
                </SelectContent>
              </Select>
            </div>
            <div className="flex gap-2">
              <Button type="submit" data-testid="inc-create" disabled={createMut.isPending || !title.trim()}>
                {createMut.isPending ? 'Creating…' : 'Create'}
              </Button>
              <Button type="button" variant="secondary" onClick={() => setShowCreate(false)}>Cancel</Button>
            </div>
          </form>
        </Card>
      )}

      <PatternCards />

      <Card className="p-0">
        {isPending ? (
          <div className="p-4"><TableSkeleton rows={5} cols={4} /></div>
        ) : error ? (
          <div className="p-4"><ErrorState message="Failed to load incidents" onRetry={() => refetch()} /></div>
        ) : list.length === 0 ? (
          <div className="p-4"><EmptyState title="No incidents" description="Reported incidents will appear here." /></div>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Title</TableHead>
                <TableHead>Severity</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Created</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {list.map(inc => (
                <TableRow key={inc.id} className="cursor-pointer" onClick={() => nav(`/incidents/${inc.id}`)}>
                  <TableCell className="font-medium">{inc.title}</TableCell>
                  <TableCell><StatusBadge tone={sevTone(inc.severity)}>{inc.severity.toUpperCase()}</StatusBadge></TableCell>
                  <TableCell><StatusBadge tone={statusTone(inc.status)}>{inc.status.replace('_', ' ')}</StatusBadge></TableCell>
                  <TableCell className="text-muted-foreground">{new Date(inc.created_at).toLocaleDateString()}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </Card>
    </div>
  );
}
