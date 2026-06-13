import { useEffect, useState } from 'react';
import { api, type AuditEvent } from '../../api/client';

export default function AdminAudit() {
  const [events, setEvents] = useState<AuditEvent[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    api.listAudit().then(setEvents).catch(() => {}).finally(() => setLoading(false));
  }, []);

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-bold">Audit Log</h1>
      {loading ? <p className="text-[var(--text-muted)]">Loading...</p> : (
        <div className="card">
          <table className="w-full text-sm">
            <thead><tr className="text-left text-[var(--text-muted)] border-b border-[var(--border)]">
              <th className="pb-2">Time</th><th className="pb-2">Actor</th><th className="pb-2">Action</th><th className="pb-2">Entity</th>
            </tr></thead>
            <tbody>
              {events.map((e: any) => (
                <tr key={e.id || e.created_at} className="border-b border-[var(--border)]">
                  <td className="py-2 text-[var(--text-muted)]">{new Date(e.created_at).toLocaleString()}</td>
                  <td className="py-2">{e.actor_id?.slice(0,8) || 'system'}…</td>
                  <td className="py-2">{e.action}</td>
                  <td className="py-2">{e.entity_type}/{e.entity_id?.slice(0,8)}…</td>
                </tr>
              ))}
              {!events.length && <tr><td colSpan={4} className="py-4 text-center text-[var(--text-muted)]">No events</td></tr>}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
