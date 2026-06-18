import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, X, Ban } from 'lucide-react';
import { api } from '@/api/client';
import { keys } from '@/api/keys';
import { Perm } from '@/auth/permissions';
import { useAuth } from '@/auth/context';
import { Card } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { StatusBadge } from '@/components/ui/status-badge';
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '@/components/ui/select';
import {
  Table, TableHeader, TableBody, TableHead, TableRow, TableCell,
} from '@/components/ui/table';
import { notify } from '@/components/Toaster';
import { InlineSpinner, ErrorState, EmptyState, CardGridSkeleton } from '@/components/PageState';
import { AUTONOMY_LEVELS, AUTONOMY_DESCRIPTIONS, AutonomyBadge } from './autonomy';
import { cn } from '@/lib/utils';

function runStatusTone(status: string): 'success' | 'danger' | 'warning' | 'info' {
  if (status === 'completed') return 'success';
  if (status === 'failed') return 'danger';
  if (status === 'running') return 'info';
  return 'warning';
}
function intentionStatusTone(status: string): 'success' | 'danger' | 'warning' | 'info' {
  if (status === 'executed') return 'success';
  if (status === 'blocked') return 'danger';
  if (status === 'pending') return 'warning';
  return 'info';
}

export function AgentsPage() {
  const { hasPermission, activeTeamId } = useAuth();
  const teamId = activeTeamId ?? '';
  const queryClient = useQueryClient();
  const [selected, setSelected] = useState<string | null>(null);
  const [viewedRun, setViewedRun] = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [createName, setCreateName] = useState('');
  const [createDesc, setCreateDesc] = useState('');
  const [createAutonomy, setCreateAutonomy] = useState('A3');

  // Grant-create form state
  const [grantTool, setGrantTool] = useState('');
  const [grantAutonomy, setGrantAutonomy] = useState('A3');
  const [grantApproval, setGrantApproval] = useState(false);

  const canRead = hasPermission(Perm.AgentsRead);

  const agentsQ = useQuery({
    queryKey: keys.agents.list(teamId),
    queryFn: ({ signal }) => api.listAgents(signal),
    enabled: !!activeTeamId && canRead,
  });
  const runsQ = useQuery({
    queryKey: keys.agentRuns.list(teamId),
    queryFn: () => api.listRuns(),
    enabled: !!activeTeamId && canRead,
  });
  const grantsQ = useQuery({
    queryKey: keys.agents.grants(teamId, selected ?? ''),
    queryFn: () => api.listGrants(selected!),
    enabled: !!selected,
  });
  const intentionsQ = useQuery({
    queryKey: keys.agentRuns.intentions(teamId, viewedRun ?? ''),
    queryFn: () => api.listIntentions(viewedRun!),
    enabled: !!viewedRun,
  });

  const invalidateAgents = () => queryClient.invalidateQueries({ queryKey: keys.agents.list(teamId) });
  const invalidateGrants = () => queryClient.invalidateQueries({ queryKey: keys.agents.grants(teamId, selected ?? '') });
  const invalidateIntentions = () => queryClient.invalidateQueries({ queryKey: keys.agentRuns.intentions(teamId, viewedRun ?? '') });

  const createMut = useMutation({
    mutationFn: () => api.createAgent({ name: createName, max_autonomy: createAutonomy, description: createDesc }),
    onSuccess: () => {
      notify.success('Agent created');
      invalidateAgents();
      setShowCreate(false);
      setCreateName(''); setCreateDesc(''); setCreateAutonomy('A3');
    },
    onError: (err) => notify.mutationError('Create agent', err),
  });

  const disableMut = useMutation({
    mutationFn: (id: string) => api.disableAgent(id),
    onSuccess: () => {
      notify.success('Agent disabled');
      invalidateAgents();
      setSelected(null);
    },
    onError: (err) => notify.mutationError('Disable agent', err),
  });

  const createGrantMut = useMutation({
    mutationFn: () => api.createGrant(selected!, { tool_name: grantTool, max_autonomy_level: grantAutonomy, requires_approval: grantApproval }),
    onSuccess: () => {
      notify.success('Grant added');
      invalidateGrants();
      setGrantTool(''); setGrantAutonomy('A3'); setGrantApproval(false);
    },
    onError: (err) => notify.mutationError('Add grant', err),
  });

  const revokeGrantMut = useMutation({
    mutationFn: (grantId: string) => api.revokeGrant(selected!, grantId),
    onSuccess: () => {
      notify.success('Grant revoked');
      invalidateGrants();
    },
    onError: (err) => notify.mutationError('Revoke grant', err),
  });

  if (!canRead) {
    return <div className="p-8"><EmptyState title="Access denied" description="You don't have permission to view agents." /></div>;
  }
  if (agentsQ.isPending) {
    return (
      <div className="space-y-6">
        <h1 className="font-heading text-2xl font-semibold tracking-tight">Agent Console</h1>
        <CardGridSkeleton count={3} />
      </div>
    );
  }

  const agents = agentsQ.data ?? [];
  const runs = runsQ.data ?? [];
  const grants = grantsQ.data ?? [];
  const intentions = intentionsQ.data ?? [];
  const selectedAgent = agents.find(a => a.id === selected);

  const selectAgent = (id: string) => {
    setSelected(id);
    setViewedRun(null);
  };
  const viewIntentions = (runId: string) => {
    setViewedRun(runId);
    // refetch happens via the enabled query; ensure fresh.
    invalidateIntentions();
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="font-heading text-2xl font-semibold tracking-tight">Agent Console</h1>
          <p className="text-sm text-muted-foreground">Agents operate under the A0–A4 autonomy ladder (A5 is policy-disabled).</p>
        </div>
        {hasPermission(Perm.AgentsCreate) && (
          <Button size="sm" variant={showCreate ? 'secondary' : 'default'} onClick={() => setShowCreate(s => !s)}>
            {showCreate ? <><X className="size-4" /> Cancel</> : <><Plus className="size-4" /> Create Agent</>}
          </Button>
        )}
      </div>

      {showCreate && (
        <Card className="space-y-4 p-5">
          <h2 className="font-heading text-sm font-semibold">New Agent</h2>
          <form className="space-y-4" onSubmit={(e) => { e.preventDefault(); if (createName.trim()) createMut.mutate(); }}>
            <div className="space-y-1.5">
              <Label htmlFor="agent-name">Name</Label>
              <Input id="agent-name" data-testid="agent-name" value={createName} onChange={e => setCreateName(e.target.value)} required />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="agent-desc">Description</Label>
              <Textarea id="agent-desc" data-testid="agent-desc" value={createDesc} onChange={e => setCreateDesc(e.target.value)} rows={2} />
            </div>
            <div className="space-y-1.5">
              <Label>Max autonomy level</Label>
              <Select value={createAutonomy} onValueChange={(v) => setCreateAutonomy(v ?? 'A3')}>
                <SelectTrigger data-testid="agent-autonomy"><SelectValue /></SelectTrigger>
                <SelectContent>
                  {AUTONOMY_LEVELS.map(a => (
                    <SelectItem key={a} value={a} disabled={a === 'A5'}>
                      {AUTONOMY_DESCRIPTIONS[a]}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="flex gap-2">
              <Button type="submit" data-testid="agent-create" disabled={createMut.isPending || !createName.trim()}>
                {createMut.isPending ? 'Creating…' : 'Create'}
              </Button>
              <Button type="button" variant="secondary" onClick={() => setShowCreate(false)}>Cancel</Button>
            </div>
          </form>
        </Card>
      )}

      <div className="grid gap-4 lg:grid-cols-3">
        {/* Agent List */}
        <div className="space-y-2">
          <h2 className="font-heading text-sm font-semibold">Agents</h2>
          {agentsQ.error ? (
            <ErrorState message="Failed to load agents" onRetry={() => agentsQ.refetch()} />
          ) : agents.length === 0 ? (
            <EmptyState title="No agents" />
          ) : (
            <ul className="space-y-2">
              {agents.map(a => (
                <li key={a.id}>
                  <button
                    type="button"
                    onClick={() => selectAgent(a.id)}
                    data-testid={`agent-row-${a.id}`}
                    className={cn(
                      'w-full rounded-lg border p-3 text-left transition-colors hover:bg-accent/50',
                      selected === a.id ? 'border-primary bg-primary/5' : 'border-border bg-card',
                    )}
                  >
                    <div className="font-medium">{a.name}</div>
                    <div className="mt-1 flex items-center gap-2">
                      <StatusBadge tone={a.status === 'active' ? 'success' : 'neutral'}>{a.status}</StatusBadge>
                      <AutonomyBadge level={a.max_autonomy} />
                    </div>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>

        {/* Detail column */}
        <div className="space-y-4 lg:col-span-2">
          {selectedAgent ? (
            <>
              <Card className="p-5">
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0">
                    <h2 className="font-heading text-lg font-semibold">{selectedAgent.name}</h2>
                    <p className="text-sm text-muted-foreground">{selectedAgent.description || 'No description'}</p>
                    <div className="mt-2 flex items-center gap-2">
                      <StatusBadge tone={selectedAgent.status === 'active' ? 'success' : 'neutral'}>{selectedAgent.status}</StatusBadge>
                      <AutonomyBadge level={selectedAgent.max_autonomy} />
                      <span className="text-xs text-muted-foreground">ID: {selectedAgent.id.slice(0, 8)}…</span>
                    </div>
                  </div>
                  {hasPermission(Perm.AgentsDisable) && selectedAgent.status === 'active' && (
                    <Button size="sm" variant="destructive" data-testid="agent-disable" onClick={() => { if (confirm('Disable this agent?')) disableMut.mutate(selectedAgent.id); }}>
                      <Ban className="size-4" /> Disable
                    </Button>
                  )}
                </div>
              </Card>

              {/* Grants */}
              <Card className="p-5">
                <h3 className="mb-3 font-heading text-sm font-semibold">Tool Grants</h3>
                {hasPermission(Perm.AgentsGrantsCreate) && (
                  <form
                    className="mb-3 flex flex-wrap items-end gap-2"
                    onSubmit={(e) => { e.preventDefault(); if (grantTool.trim()) createGrantMut.mutate(); }}
                  >
                    <div className="min-w-[160px] flex-1 space-y-1">
                      <Label htmlFor="grant-tool" className="text-xs">Tool</Label>
                      <Input id="grant-tool" data-testid="grant-tool" value={grantTool} onChange={e => setGrantTool(e.target.value)} required />
                    </div>
                    <div className="w-28 space-y-1">
                      <Label className="text-xs">Autonomy</Label>
                      <Select value={grantAutonomy} onValueChange={(v) => setGrantAutonomy(v ?? 'A3')}>
                        <SelectTrigger data-testid="grant-autonomy" className="h-8 text-xs"><SelectValue /></SelectTrigger>
                        <SelectContent>
                          {AUTONOMY_LEVELS.map(a => <SelectItem key={a} value={a} disabled={a === 'A5'}>{a}</SelectItem>)}
                        </SelectContent>
                      </Select>
                    </div>
                    <label className="flex h-8 items-center gap-1.5 text-xs">
                      <input type="checkbox" data-testid="grant-approval" checked={grantApproval} onChange={e => setGrantApproval(e.target.checked)} />
                      Approval
                    </label>
                    <Button type="submit" size="sm" data-testid="grant-add" disabled={createGrantMut.isPending || !grantTool.trim()}>
                      Add
                    </Button>
                  </form>
                )}
                {grantsQ.isPending ? <InlineSpinner /> : grants.length === 0 ? (
                  <p className="text-sm text-muted-foreground">No grants</p>
                ) : (
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Tool</TableHead><TableHead>Autonomy</TableHead><TableHead>Approval</TableHead><TableHead>Status</TableHead><TableHead />
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {grants.map(g => (
                        <TableRow key={g.id}>
                          <TableCell className="font-medium">{g.tool_name}</TableCell>
                          <TableCell><AutonomyBadge level={g.max_autonomy_level} /></TableCell>
                          <TableCell>{g.requires_approval ? 'Yes' : 'No'}</TableCell>
                          <TableCell>
                            {g.revoked_at ? <StatusBadge tone="neutral">Revoked</StatusBadge> : <StatusBadge tone="success">Active</StatusBadge>}
                          </TableCell>
                          <TableCell>
                            {!g.revoked_at && hasPermission(Perm.AgentsGrantsRevoke) && (
                              <Button size="sm" variant="ghost" className="text-destructive" onClick={() => revokeGrantMut.mutate(g.id)}>Revoke</Button>
                            )}
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                )}
              </Card>
            </>
          ) : (
            <EmptyState title="Select an agent" description="Choose an agent from the list to view its details and grants." />
          )}

          {/* Runs */}
          <Card className="p-5">
            <h3 className="mb-3 font-heading text-sm font-semibold">Agent Runs</h3>
            {runsQ.isPending ? <InlineSpinner /> : runs.length === 0 ? (
              <p className="text-sm text-muted-foreground">No runs</p>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow><TableHead>ID</TableHead><TableHead>Status</TableHead><TableHead>Created</TableHead><TableHead /></TableRow>
                </TableHeader>
                <TableBody>
                  {runs.slice(0, 10).map(r => (
                    <TableRow key={r.id} className={viewedRun === r.id ? 'bg-accent/50' : ''}>
                      <TableCell className="font-mono text-xs">{r.id.slice(0, 8)}…</TableCell>
                      <TableCell><StatusBadge tone={runStatusTone(r.status)}>{r.status}</StatusBadge></TableCell>
                      <TableCell className="text-muted-foreground">{new Date(r.created_at).toLocaleString()}</TableCell>
                      <TableCell>
                        <Button size="sm" variant="ghost" onClick={() => viewIntentions(r.id)}>View</Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </Card>

          {/* Intentions */}
          {viewedRun && (
            <Card className="p-5">
              <h3 className="mb-3 font-heading text-sm font-semibold">Intentions — run {viewedRun.slice(0, 8)}…</h3>
              {intentionsQ.isPending ? <InlineSpinner /> : intentions.length === 0 ? (
                <p className="text-sm text-muted-foreground">No intentions for this run.</p>
              ) : (
                <Table>
                  <TableHeader>
                    <TableRow><TableHead>Tool</TableHead><TableHead>Autonomy</TableHead><TableHead>Status</TableHead><TableHead>Reason</TableHead></TableRow>
                  </TableHeader>
                  <TableBody>
                    {intentions.map(i => (
                      <TableRow key={i.id}>
                        <TableCell className="font-medium">{i.requested_tool}</TableCell>
                        <TableCell><AutonomyBadge level={i.autonomy_level} /></TableCell>
                        <TableCell><StatusBadge tone={intentionStatusTone(i.status)}>{i.status}</StatusBadge></TableCell>
                        <TableCell className="text-muted-foreground">{i.blocked_reason || '—'}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              )}
            </Card>
          )}
        </div>
      </div>
    </div>
  );
}

export default AgentsPage;
