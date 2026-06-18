import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, X, Copy, Check, KeyRound, AlertTriangle, RotateCw, Ban } from 'lucide-react';
import { api, getStoredTeamId } from '@/api/client';
import { keys } from '@/api/keys';
import { Card } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { StatusBadge } from '@/components/ui/status-badge';
import {
  Table, TableHeader, TableBody, TableHead, TableRow, TableCell,
} from '@/components/ui/table';
import { notify } from '@/components/Toaster';
import { TableSkeleton, ErrorState, EmptyState } from '@/components/PageState';

interface CreatedSecret { key: string; signing_secret: string; id: string }

function keyStatus(k: Record<string, unknown>) {
  if (k.revoked_at) return { label: 'Revoked', tone: 'danger' as const };
  if (k.rotation_required) return { label: 'Rotation Required', tone: 'warning' as const };
  const expiresAt = k.expires_at as string | undefined;
  if (expiresAt && new Date(expiresAt) < new Date()) return { label: 'Expired', tone: 'danger' as const };
  return { label: 'Active', tone: 'success' as const };
}

export default function AdminIntegrations() {
  const teamId = getStoredTeamId();
  const queryClient = useQueryClient();
  const [showCreate, setShowCreate] = useState(false);
  const [createdKey, setCreatedKey] = useState<CreatedSecret | null>(null);
  const [copied, setCopied] = useState('');
  const [name, setName] = useState('');
  const [sources, setSources] = useState('');
  const [scopes, setScopes] = useState('');
  const [allowUnsigned, setAllowUnsigned] = useState(false);

  const keysQ = useQuery({
    queryKey: keys.integrationKeys(teamId ?? ''),
    queryFn: () => api.listIntegrationKeys(teamId!),
    enabled: !!teamId,
  });

  const invalidate = () => queryClient.invalidateQueries({ queryKey: keys.integrationKeys(teamId ?? '') });

  const createMut = useMutation({
    mutationFn: () => api.createIntegrationKey(teamId!, {
      name,
      allowed_sources: sources.split(',').map(s => s.trim()).filter(Boolean),
      allowed_scopes: scopes.split(',').map(s => s.trim()).filter(Boolean),
      allow_unsigned_dev: allowUnsigned,
    }),
    onSuccess: (result) => {
      // Secret is shown ONCE — surface the reveal modal immediately.
      setCreatedKey({ key: result.key, signing_secret: result.signing_secret, id: result.id });
      setShowCreate(false);
      setName(''); setSources(''); setScopes(''); setAllowUnsigned(false);
      invalidate();
      notify.success('Integration key created');
    },
    onError: (err) => notify.mutationError('Create key', err),
  });

  const revokeMut = useMutation({
    mutationFn: (keyId: string) => api.revokeIntegrationKey(teamId!, keyId),
    onSuccess: () => { invalidate(); notify.success('Key revoked'); },
    onError: (err) => notify.mutationError('Revoke key', err),
  });

  const rotateMut = useMutation({
    mutationFn: (keyId: string) => api.rotateIntegrationKey(teamId!, keyId),
    onSuccess: (result) => {
      setCreatedKey({ key: result.key, signing_secret: result.signing_secret, id: result.id });
      invalidate();
      notify.success('Key rotated');
    },
    onError: (err) => notify.mutationError('Rotate key', err),
  });

  const copyToClipboard = (text: string, label: string) => {
    navigator.clipboard.writeText(text);
    setCopied(label);
    setTimeout(() => setCopied(''), 2000);
  };

  if (keysQ.isPending) {
    return (
      <div className="space-y-6">
        <h1 className="font-heading text-2xl font-semibold tracking-tight">Integration Management</h1>
        <Card className="p-4"><TableSkeleton rows={4} cols={7} /></Card>
      </div>
    );
  }
  if (keysQ.error) {
    return (
      <div className="space-y-6">
        <h1 className="font-heading text-2xl font-semibold tracking-tight">Integration Management</h1>
        <ErrorState message="Failed to load integration keys" onRetry={() => keysQ.refetch()} />
      </div>
    );
  }

  const list = keysQ.data ?? [];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="font-heading text-2xl font-semibold tracking-tight">Integration Management</h1>
        <Button size="sm" variant={showCreate ? 'secondary' : 'default'} onClick={() => setShowCreate(s => !s)}>
          {showCreate ? <><X className="size-4" /> Cancel</> : <><Plus className="size-4" /> Create Key</>}
        </Button>
      </div>

      {/* Created-secret reveal modal — shown once after create/rotate */}
      {createdKey && (
        <Card className="border-2 border-warning/50 p-5">
          <div className="mb-3 flex items-center gap-2">
            <AlertTriangle className="size-5 text-warning" />
            <h3 className="font-heading text-sm font-semibold text-warning">Save These Credentials — Shown Only Once</h3>
          </div>
          <p className="mb-3 text-sm text-muted-foreground">Copy and store securely. These cannot be retrieved again.</p>
          <div className="space-y-2">
            <div>
              <Label className="text-xs">Integration Key</Label>
              <div className="flex gap-2">
                <code className="flex-1 break-all rounded bg-muted p-2 text-xs">{createdKey.key}</code>
                <Button size="sm" variant="secondary" onClick={() => copyToClipboard(createdKey.key, 'key')}>
                  {copied === 'key' ? <Check className="size-4" /> : <Copy className="size-4" />}
                </Button>
              </div>
            </div>
            <div>
              <Label className="text-xs">Signing Secret</Label>
              <div className="flex gap-2">
                <code className="flex-1 break-all rounded bg-muted p-2 text-xs">{createdKey.signing_secret}</code>
                <Button size="sm" variant="secondary" onClick={() => copyToClipboard(createdKey.signing_secret, 'secret')}>
                  {copied === 'secret' ? <Check className="size-4" /> : <Copy className="size-4" />}
                </Button>
              </div>
            </div>
          </div>
          <Button className="mt-3" size="sm" variant="secondary" onClick={() => setCreatedKey(null)}>I've saved them</Button>
        </Card>
      )}

      {showCreate && (
        <Card className="space-y-4 p-5">
          <h3 className="font-heading text-sm font-semibold">Create Integration Key</h3>
          <div className="space-y-1.5">
            <Label htmlFor="ik-name">Name</Label>
            <Input id="ik-name" data-testid="ik-name" value={name} onChange={e => setName(e.target.value)} placeholder="e.g. Grafana Alerts" />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="ik-sources">Allowed Sources (comma-separated)</Label>
            <Input id="ik-sources" data-testid="ik-sources" value={sources} onChange={e => setSources(e.target.value)} placeholder="grafana, prometheus, or *" />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="ik-scopes">Allowed Scopes (comma-separated)</Label>
            <Input id="ik-scopes" data-testid="ik-scopes" value={scopes} onChange={e => setScopes(e.target.value)} placeholder="webhooks:ingest, alerts:create, or *" />
          </div>
          <label className="flex items-center gap-2 text-sm">
            <input type="checkbox" data-testid="ik-unsigned" checked={allowUnsigned} onChange={e => setAllowUnsigned(e.target.checked)} />
            Allow unsigned webhooks (dev only)
          </label>
          <Button data-testid="ik-create" disabled={createMut.isPending || !name || !sources || !scopes} onClick={() => createMut.mutate()}>
            <KeyRound className="size-4" /> {createMut.isPending ? 'Creating…' : 'Create Key'}
          </Button>
        </Card>
      )}

      <div>
        <h2 className="mb-2 font-heading text-lg font-semibold">Integration Keys ({list.length})</h2>
        <Card className="p-0">
          {list.length === 0 ? (
            <div className="p-4"><EmptyState title="No integration keys yet" /></div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead><TableHead>Prefix</TableHead><TableHead>Sources</TableHead><TableHead>Scopes</TableHead><TableHead>Status</TableHead><TableHead>Created</TableHead><TableHead>Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {list.map(k => {
                  const st = keyStatus(k);
                  return (
                    <TableRow key={k.id} className={k.revoked_at ? 'opacity-50' : ''}>
                      <TableCell className="font-medium">{k.name}</TableCell>
                      <TableCell className="font-mono text-xs">{k.prefix}…</TableCell>
                      <TableCell className="text-xs">{Array.isArray(k.allowed_sources) ? k.allowed_sources.join(', ') : String(k.allowed_sources ?? '')}</TableCell>
                      <TableCell className="text-xs">{Array.isArray(k.allowed_scopes) ? k.allowed_scopes.join(', ') : String(k.allowed_scopes ?? '')}</TableCell>
                      <TableCell><StatusBadge tone={st.tone}>{st.label}</StatusBadge></TableCell>
                      <TableCell className="text-xs text-muted-foreground">{k.created_at ? new Date(k.created_at).toLocaleDateString() : '—'}</TableCell>
                      <TableCell>
                        {!k.revoked_at && (
                          <div className="flex gap-2">
                            <Button size="sm" variant="ghost" className="text-warning" data-testid={`ik-rotate-${k.id}`} onClick={() => { if (confirm('Rotate this key? The old key is revoked immediately.')) rotateMut.mutate(k.id); }}>
                              <RotateCw className="size-3.5" /> Rotate
                            </Button>
                            <Button size="sm" variant="ghost" className="text-destructive" data-testid={`ik-revoke-${k.id}`} onClick={() => { if (confirm('Revoke this integration key? This cannot be undone.')) revokeMut.mutate(k.id); }}>
                              <Ban className="size-3.5" /> Revoke
                            </Button>
                          </div>
                        )}
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          )}
        </Card>
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        <Card className="p-5">
          <h3 className="mb-2 font-heading text-sm font-semibold">Webhook Signing</h3>
          <p className="text-sm text-muted-foreground">
            New keys include a signing secret for HMAC-SHA256 webhook verification. Keys marked
            <StatusBadge tone="warning">Rotation Required</StatusBadge> lack signing secrets and must be rotated for production.
          </p>
        </Card>
        <Card className="p-5">
          <h3 className="mb-2 font-heading text-sm font-semibold">Proxmox Integration</h3>
          <p className="text-sm text-muted-foreground">
            Status: Configure via <code className="rounded bg-muted px-1">PROXMOX_ENABLED</code> env var. Mode: read-only (no mutation endpoints).
          </p>
        </Card>
        <Card className="p-5">
          <h3 className="mb-2 font-heading text-sm font-semibold">Email Delivery</h3>
          <p className="text-sm text-muted-foreground">
            Configure via <code className="rounded bg-muted px-1">EMAIL_MODE</code> (dev/smtp/disabled) and <code className="rounded bg-muted px-1">SMTP_*</code> env vars.
          </p>
        </Card>
      </div>
    </div>
  );
}
