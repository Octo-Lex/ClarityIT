import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { api, type Incident } from '../../api/client';
import { useAuth } from '../../auth/context';
import PatternCards from './PatternCards';

export default function IncidentList() {
  const { activeTeamId, hasPermission } = useAuth();
  const nav = useNavigate();
  const [incidents, setIncidents] = useState<Incident[]>([]);
  const [showCreate, setShowCreate] = useState(false);
  const [title, setTitle] = useState('');
  const [severity, setSeverity] = useState('sev3');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!activeTeamId) return;
    setLoading(true);
    api.listIncidents().then(setIncidents).catch(() => {}).finally(() => setLoading(false));
  }, [activeTeamId]);

  const create = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true); setError('');
    try {
      const r = await api.createIncident({ title, severity });
      setShowCreate(false); setTitle(''); setSeverity('sev3');
      api.listIncidents().then(setIncidents).finally(() => setLoading(false));
    } catch (e: any) { setError(e.message); setLoading(false); }
  };

  const sevColor: Record<string, string> = { sev1: 'badge-red', sev2: 'badge-yellow', sev3: 'badge-blue', sev4: 'badge-gray' };

  return (
    <div className="space-y-4">
      <div className="flex justify-between items-center">
        <h1 className="text-2xl font-bold">Incidents</h1>
        {hasPermission('incidents.create') && <button onClick={() => setShowCreate(!showCreate)} className="text-sm">+ New Incident</button>}
      </div>

      {showCreate && (
        <form onSubmit={create} className="card space-y-3">
          {error && <div className="error-msg">{error}</div>}
          <input placeholder="Title *" value={title} onChange={e => setTitle(e.target.value)} required />
          <select value={severity} onChange={e => setSeverity(e.target.value)}>
            <option value="sev1">SEV1 — Critical</option><option value="sev2">SEV2 — Major</option>
            <option value="sev3">SEV3 — Minor</option><option value="sev4">SEV4 — Informational</option>
          </select>
          <div className="flex gap-2"><button type="submit">Create</button><button type="button" className="btn-secondary" onClick={() => setShowCreate(false)}>Cancel</button></div>
        </form>
      )}

      {loading && !incidents.length ? <p className="text-[var(--text-muted)]">Loading...</p> : (
        <>
        {/* v1.2 Track 2: Incident Pattern Cards */}
        <PatternCards />

        <div className="card">
          <table className="w-full text-sm">
            <thead><tr className="text-left text-[var(--text-muted)] border-b border-[var(--border)]">
              <th className="pb-2">Title</th><th className="pb-2">Severity</th><th className="pb-2">Status</th><th className="pb-2">Created</th>
            </tr></thead>
            <tbody>
              {incidents.map(inc => (
                <tr key={inc.id} className="border-b border-[var(--border)] hover:bg-[var(--border)] cursor-pointer" onClick={() => nav(`/incidents/${inc.id}`)}>
                  <td className="py-2 font-medium">{inc.title}</td>
                  <td className="py-2"><span className={`badge ${sevColor[inc.severity] || 'badge-gray'}`}>{inc.severity}</span></td>
                  <td className="py-2"><span className={`badge ${inc.status === 'resolved' ? 'badge-green' : 'badge-blue'}`}>{inc.status}</span></td>
                  <td className="py-2 text-[var(--text-muted)]">{new Date(inc.created_at).toLocaleDateString()}</td>
                </tr>
              ))}
              {!incidents.length && <tr><td colSpan={4} className="py-4 text-center text-[var(--text-muted)]">No incidents</td></tr>}
            </tbody>
          </table>
        </div>
        </>
      )}
    </div>
  );
}
