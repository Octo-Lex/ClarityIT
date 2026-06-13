import { useEffect, useState, useCallback } from 'react';
import { api } from '../../api/client';

export default function AdminIntegrations() {
  const [keys, setKeys] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [showCreate, setShowCreate] = useState(false);
  const [createdKey, setCreatedKey] = useState<{ key: string; signing_secret: string; id: string } | null>(null);
  const [copied, setCopied] = useState('');

  // Create form state
  const [name, setName] = useState('');
  const [sources, setSources] = useState('');
  const [scopes, setScopes] = useState('');
  const [allowUnsigned, setAllowUnsigned] = useState(false);
  const [creating, setCreating] = useState(false);

  const teamId = typeof localStorage !== 'undefined' ? localStorage.getItem('clarityit_team') : null;

  const load = useCallback(async () => {
    if (!teamId) { setError('No active team'); return; }
    try {
      const data = await api.listIntegrationKeys(teamId);
      setKeys(data);
      setError('');
    } catch (e: any) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  }, [teamId]);

  useEffect(() => { load(); }, [load]);

  const handleCreate = async () => {
    if (!teamId) return;
    setCreating(true);
    try {
      const result = await api.createIntegrationKey(teamId, {
        name,
        allowed_sources: sources.split(',').map(s => s.trim()).filter(Boolean),
        allowed_scopes: scopes.split(',').map(s => s.trim()).filter(Boolean),
        allow_unsigned_dev: allowUnsigned,
      });
      setCreatedKey(result);
      setShowCreate(false);
      setName(''); setSources(''); setScopes(''); setAllowUnsigned(false);
      load();
    } catch (e: any) {
      setError(e.message);
    } finally {
      setCreating(false);
    }
  };

  const handleRevoke = async (keyId: string) => {
    if (!teamId || !confirm('Revoke this integration key? This cannot be undone.')) return;
    try {
      await api.revokeIntegrationKey(teamId, keyId);
      load();
    } catch (e: any) {
      setError(e.message);
    }
  };

  const handleRotate = async (keyId: string) => {
    if (!teamId || !confirm('Rotate this key? The old key will be revoked immediately and a new key + signing secret will be generated.')) return;
    try {
      const result = await api.rotateIntegrationKey(teamId, keyId);
      setCreatedKey({ key: result.key, signing_secret: result.signing_secret, id: result.id });
      load();
    } catch (e: any) {
      setError(e.message);
    }
  };

  const copyToClipboard = (text: string, label: string) => {
    navigator.clipboard.writeText(text);
    setCopied(label);
    setTimeout(() => setCopied(''), 2000);
  };

  if (loading) return <div className="text-[var(--text-muted)]">Loading integrations...</div>;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Integration Management</h1>
        <button onClick={() => setShowCreate(!showCreate)} className="px-3 py-1.5 rounded bg-[var(--primary)] text-white text-sm">
          {showCreate ? 'Cancel' : '+ Create Key'}
        </button>
      </div>

      {error && <div className="card p-3 text-[var(--danger)]">{error}</div>}

      {/* Created Key Modal */}
      {createdKey && (
        <div className="card p-4 border-2 border-[var(--warning)]">
          <h3 className="text-lg font-bold text-[var(--warning)]">⚠️ Save These Credentials — Shown Only Once</h3>
          <p className="text-sm text-[var(--text-muted)] mt-1 mb-3">Copy and store securely. These cannot be retrieved again.</p>
          <div className="space-y-2">
            <div>
              <label className="text-xs text-[var(--text-muted)]">Integration Key</label>
              <div className="flex gap-2">
                <code className="flex-1 p-2 bg-[var(--border)] rounded text-xs break-all">{createdKey.key}</code>
                <button onClick={() => copyToClipboard(createdKey.key, 'key')} className="px-2 py-1 text-xs bg-[var(--card)] rounded border border-[var(--border)]">
                  {copied === 'key' ? '✓' : 'Copy'}
                </button>
              </div>
            </div>
            <div>
              <label className="text-xs text-[var(--text-muted)]">Signing Secret</label>
              <div className="flex gap-2">
                <code className="flex-1 p-2 bg-[var(--border)] rounded text-xs break-all">{createdKey.signing_secret}</code>
                <button onClick={() => copyToClipboard(createdKey.signing_secret, 'secret')} className="px-2 py-1 text-xs bg-[var(--card)] rounded border border-[var(--border)]">
                  {copied === 'secret' ? '✓' : 'Copy'}
                </button>
                </div>
            </div>
          </div>
          <button onClick={() => setCreatedKey(null)} className="mt-3 px-3 py-1 text-sm bg-[var(--border)] rounded">I've saved them</button>
        </div>
      )}

      {/* Create Form */}
      {showCreate && (
        <div className="card p-4 space-y-3">
          <h3 className="font-semibold">Create Integration Key</h3>
          <div>
            <label className="text-xs text-[var(--text-muted)]">Name</label>
            <input value={name} onChange={e => setName(e.target.value)} placeholder="e.g. Grafana Alerts"
              className="w-full mt-1 p-2 bg-[var(--border)] rounded text-sm" />
          </div>
          <div>
            <label className="text-xs text-[var(--text-muted)]">Allowed Sources (comma-separated)</label>
            <input value={sources} onChange={e => setSources(e.target.value)} placeholder="grafana, prometheus, or *"
              className="w-full mt-1 p-2 bg-[var(--border)] rounded text-sm" />
          </div>
          <div>
            <label className="text-xs text-[var(--text-muted)]">Allowed Scopes (comma-separated)</label>
            <input value={scopes} onChange={e => setScopes(e.target.value)} placeholder="webhooks:ingest, alerts:create, or *"
              className="w-full mt-1 p-2 bg-[var(--border)] rounded text-sm" />
          </div>
          <label className="flex items-center gap-2 text-sm">
            <input type="checkbox" checked={allowUnsigned} onChange={e => setAllowUnsigned(e.target.checked)} />
            Allow unsigned webhooks (dev only)
          </label>
          <button onClick={handleCreate} disabled={creating || !name || !sources || !scopes}
            className="px-4 py-2 rounded bg-[var(--primary)] text-white text-sm disabled:opacity-50">
            {creating ? 'Creating...' : 'Create Key'}
          </button>
        </div>
      )}

      {/* Keys Table */}
      <div>
        <h2 className="text-lg font-semibold mb-2">Integration Keys ({keys.length})</h2>
        <div className="card overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left text-[var(--text-muted)] border-b border-[var(--border)]">
                <th className="pb-2 pr-4">Name</th>
                <th className="pb-2 pr-4">Prefix</th>
                <th className="pb-2 pr-4">Sources</th>
                <th className="pb-2 pr-4">Scopes</th>
                <th className="pb-2 pr-4">Status</th>
                <th className="pb-2 pr-4">Created</th>
                <th className="pb-2">Actions</th>
              </tr>
            </thead>
            <tbody>
              {keys.map(k => (
                <tr key={k.id} className={`border-b border-[var(--border)] ${k.revoked_at ? 'opacity-50' : ''}`}>
                  <td className="py-2 pr-4 font-medium">{k.name}</td>
                  <td className="py-2 pr-4 font-mono text-xs">{k.prefix}…</td>
                  <td className="py-2 pr-4 text-xs">{Array.isArray(k.allowed_sources) ? k.allowed_sources.join(', ') : k.allowed_sources}</td>
                  <td className="py-2 pr-4 text-xs">{Array.isArray(k.allowed_scopes) ? k.allowed_scopes.join(', ') : k.allowed_scopes}</td>
                  <td className="py-2 pr-4">
                    {k.revoked_at ? (
                      <span className="text-xs text-[var(--danger)]">Revoked</span>
                    ) : k.rotation_required ? (
                      <span className="text-xs px-2 py-0.5 rounded bg-[var(--warning)] text-black font-medium">⚠ Rotation Required</span>
                    ) : k.expires_at && new Date(k.expires_at) < new Date() ? (
                      <span className="text-xs text-[var(--danger)]">Expired</span>
                    ) : (
                      <span className="text-xs text-[var(--success)]">Active</span>
                    )}
                  </td>
                  <td className="py-2 pr-4 text-xs text-[var(--text-muted)]">{k.created_at ? new Date(k.created_at).toLocaleDateString() : '—'}</td>
                  <td className="py-2 flex gap-2">
                    {!k.revoked_at && (
                      <>
                        <button onClick={() => handleRotate(k.id)} className="text-xs text-[var(--warning)] hover:underline">Rotate</button>
                        <button onClick={() => handleRevoke(k.id)} className="text-xs text-[var(--danger)] hover:underline">Revoke</button>
                      </>
                    )}
                  </td>
                </tr>
              ))}
              {!keys.length && (
                <tr><td colSpan={7} className="py-4 text-center text-[var(--text-muted)]">No integration keys yet</td></tr>
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* Integration Status */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div className="card p-4">
          <h3 className="font-semibold mb-2">Webhook Signing</h3>
          <p className="text-sm text-[var(--text-muted)]">
            New keys include a signing secret for HMAC-SHA256 webhook verification.
            Keys marked <span className="text-[var(--warning)]">⚠ Rotation Required</span> lack signing secrets
            and must be rotated for production webhook use.
          </p>
        </div>
        <div className="card p-4">
          <h3 className="font-semibold mb-2">Proxmox Integration</h3>
          <p className="text-sm text-[var(--text-muted)]">
            Status: <span className="text-[var(--text-muted)]">Configure via PROXMOX_ENABLED env var</span><br />
            Mode: read-only (no mutation endpoints)
          </p>
        </div>
        <div className="card p-4">
          <h3 className="font-semibold mb-2">Email Delivery</h3>
          <p className="text-sm text-[var(--text-muted)]">
            Configure via EMAIL_MODE (dev/smtp/disabled) and SMTP_* env vars.
            Dev mode returns preview links. SMTP mode sends real email.
          </p>
        </div>
      </div>
    </div>
  );
}
