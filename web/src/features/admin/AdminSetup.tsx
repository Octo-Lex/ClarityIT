import { useEffect, useState } from 'react';
import { api } from '../../api/client';

export default function AdminSetup() {
  const [status, setStatus] = useState<Record<string, any> | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api.setupStatus()
      .then(setStatus)
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  if (loading) return <div className="text-[var(--text-muted)]">Loading setup status...</div>;
  if (!status) return <div className="text-[var(--danger)]">Failed to load setup status</div>;

  const items: { key: string; label: string; detail?: string }[] = [
    { key: 'bootstrap_complete', label: 'Platform Bootstrapped' },
    { key: 'first_team_exists', label: 'First Team Created' },
    { key: 'users_exist', label: 'Users Exist' },
    { key: 'integration_key_created', label: 'Integration Key Created' },
    { key: 'email_configured', label: 'SMTP Email Configured', detail: `Mode: ${status.email_mode || 'dev'}` },
  ];

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Admin Setup Checklist</h1>

      <div className="space-y-2">
        {items.map(item => {
          const done = status[item.key] === true;
          return (
            <div key={item.key} className="card p-3 flex items-center gap-3">
              <span className={`text-xl ${done ? 'text-[var(--success)]' : 'text-[var(--text-muted)]'}`}>
                {done ? '✓' : '○'}
              </span>
              <div className="flex-1">
                <div className="text-sm font-medium">{item.label}</div>
                {item.detail && <div className="text-xs text-[var(--text-muted)]">{item.detail}</div>}
              </div>
              <span className={`text-xs ${done ? 'text-[var(--success)]' : 'text-[var(--warning)]'}`}>
                {done ? 'Complete' : 'Pending'}
              </span>
            </div>
          );
        })}
      </div>

      {/* Integration Status */}
      <div className="card p-4">
        <h2 className="font-semibold mb-2">Integration Status</h2>
        <div className="grid grid-cols-2 gap-3 text-sm">
          <div>
            <span className="text-[var(--text-muted)]">Proxmox Mode:</span>{' '}
            <span className="font-medium">{status.proxmox_mode}</span>
          </div>
          <div>
            <span className="text-[var(--text-muted)]">Webhook Signing:</span>{' '}
            <span className="font-medium">{status.webhook_signing_enforced ? 'Enforced (production)' : 'Optional (development)'}</span>
          </div>
        </div>
      </div>

      {/* Next Actions */}
      <div className="card p-4">
        <h2 className="font-semibold mb-2">Next Actions</h2>
        <ul className="space-y-1 text-sm">
          {(status.next_actions || []).map((action: string, i: number) => (
            <li key={i} className="text-[var(--text-muted)]">→ {action}</li>
          ))}
        </ul>
      </div>

      {/* Agent Profile Info */}
      <div className="card p-4">
        <h2 className="font-semibold mb-2">Agent Worker</h2>
        <p className="text-sm text-[var(--text-muted)]">{status.agent_profile_required}</p>
      </div>

      {/* Docs Links */}
      <div className="card p-4">
        <h2 className="font-semibold mb-2">Documentation</h2>
        <div className="text-sm space-y-1">
          <div>📘 <a href="#" className="text-[var(--primary)] underline">First Run Guide</a></div>
          <div>🔧 <a href="#" className="text-[var(--primary)] underline">Admin Runbook</a></div>
          <div>💾 <a href="#" className="text-[var(--primary)] underline">Backup & Restore</a></div>
          <div>🔒 <a href="#" className="text-[var(--primary)] underline">Security Audit</a></div>
          <div>☁️ <a href="#" className="text-[var(--primary)] underline">Cloudflare Tunnel</a></div>
        </div>
      </div>
    </div>
  );
}
