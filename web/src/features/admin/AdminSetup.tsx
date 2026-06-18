import { useQuery } from '@tanstack/react-query';
import { CheckCircle2, Circle, BookOpen, Wrench, Database, ShieldCheck, Cloud } from 'lucide-react';
import { api } from '@/api/client';
import { keys } from '@/api/keys';
import { Card } from '@/components/ui/card';
import { StatusBadge } from '@/components/ui/status-badge';
import { InlineSpinner, ErrorState } from '@/components/PageState';

export default function AdminSetup() {
  const { data: status, isPending, error, refetch } = useQuery({
    queryKey: keys.admin.setupStatus(),
    queryFn: () => api.setupStatus(),
  });

  if (isPending) return <InlineSpinner />;
  if (error) return <ErrorState message="Failed to load setup status" onRetry={() => refetch()} />;
  if (!status) return <ErrorState message="No setup status available" />;

  const items: { key: string; label: string; detail?: string }[] = [
    { key: 'bootstrap_complete', label: 'Platform Bootstrapped' },
    { key: 'first_team_exists', label: 'First Team Created' },
    { key: 'users_exist', label: 'Users Exist' },
    { key: 'integration_key_created', label: 'Integration Key Created' },
    { key: 'email_configured', label: 'SMTP Email Configured', detail: `Mode: ${status.email_mode || 'dev'}` },
  ];

  const nextActions = (status.next_actions || []) as string[];
  const docs = [
    { icon: BookOpen, label: 'First Run Guide' },
    { icon: Wrench, label: 'Admin Runbook' },
    { icon: Database, label: 'Backup & Restore' },
    { icon: ShieldCheck, label: 'Security Audit' },
    { icon: Cloud, label: 'Cloudflare Tunnel' },
  ];

  return (
    <div className="space-y-6">
      <h1 className="font-heading text-2xl font-semibold tracking-tight">Admin Setup Checklist</h1>

      <div className="space-y-2">
        {items.map(item => {
          const done = status[item.key] === true;
          const Icon = done ? CheckCircle2 : Circle;
          return (
            <Card key={item.key} className="flex items-center gap-3 p-4">
              <Icon className={`size-5 shrink-0 ${done ? 'text-success' : 'text-muted-foreground'}`} />
              <div className="flex-1">
                <div className="text-sm font-medium">{item.label}</div>
                {item.detail && <div className="text-xs text-muted-foreground">{item.detail}</div>}
              </div>
              <StatusBadge tone={done ? 'success' : 'warning'}>{done ? 'Complete' : 'Pending'}</StatusBadge>
            </Card>
          );
        })}
      </div>

      <Card className="p-5">
        <h2 className="mb-3 font-heading text-sm font-semibold">Integration Status</h2>
        <div className="grid gap-3 text-sm sm:grid-cols-2">
          <div>
            <span className="text-muted-foreground">Proxmox Mode:</span>{' '}
            <span className="font-medium">{String(status.proxmox_mode ?? '—')}</span>
          </div>
          <div>
            <span className="text-muted-foreground">Webhook Signing:</span>{' '}
            <span className="font-medium">{status.webhook_signing_enforced ? 'Enforced (production)' : 'Optional (development)'}</span>
          </div>
        </div>
      </Card>

      {nextActions.length > 0 && (
        <Card className="p-5">
          <h2 className="mb-2 font-heading text-sm font-semibold">Next Actions</h2>
          <ul className="space-y-1 text-sm">
            {nextActions.map((action, i) => (
              <li key={i} className="text-muted-foreground">→ {action}</li>
            ))}
          </ul>
        </Card>
      )}

      <Card className="p-5">
        <h2 className="mb-2 font-heading text-sm font-semibold">Agent Worker</h2>
        <p className="text-sm text-muted-foreground">{String(status.agent_profile_required ?? '—')}</p>
      </Card>

      <Card className="p-5">
        <h2 className="mb-3 font-heading text-sm font-semibold">Documentation</h2>
        <div className="grid gap-2 text-sm sm:grid-cols-2">
          {docs.map(d => {
            const Icon = d.icon;
            return (
              <a key={d.label} href="#" className="flex items-center gap-2 text-primary hover:underline">
                <Icon className="size-4" /> {d.label}
              </a>
            );
          })}
        </div>
      </Card>
    </div>
  );
}
