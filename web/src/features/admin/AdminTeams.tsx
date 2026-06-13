import { useEffect, useState } from 'react';
import { api } from '../../api/client';

export default function AdminTeams() {
  const [teams, setTeams] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    api.listTeams().then(setTeams).catch(() => {}).finally(() => setLoading(false));
  }, []);

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-bold">Teams</h1>
      {loading ? <p className="text-[var(--text-muted)]">Loading...</p> : (
        <div className="grid grid-cols-2 gap-4">
          {teams.map(t => (
            <div key={t.id} className="card">
              <h3 className="font-semibold">{t.name}</h3>
              <p className="text-sm text-[var(--text-muted)]">{t.slug}</p>
              {t.description && <p className="text-sm mt-1">{t.description}</p>}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
